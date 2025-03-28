package copa

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/containerd/platforms"
	"github.com/docker/buildx/build"
	"github.com/docker/cli/cli/config"
	"github.com/quay/claircore/osrelease"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"

	"github.com/distribution/reference"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/project-copacetic/copacetic/pkg/buildkit"
	"github.com/project-copacetic/copacetic/pkg/pkgmgr"
	"github.com/project-copacetic/copacetic/pkg/report"
	"github.com/project-copacetic/copacetic/pkg/types/unversioned"
	"github.com/project-copacetic/copacetic/pkg/utils"
	"github.com/project-copacetic/copacetic/pkg/vex"
)

// https://github.com/project-copacetic/copacetic/blob/v0.6.2/pkg/patch/patch.go

const (
	defaultPatchedTagSuffix = "patched"
	copaProduct             = "copa"
	defaultRegistry         = "docker.io"
	defaultTag              = "latest"
)

// Patch command applies package updates to an OCI image given a vulnerability report.
func Patch(ctx context.Context, timeout time.Duration, image, reportFile, patchedTag, workingFolder, scanner, format, output string, ignoreError bool, bkOpts buildkit.Opts, out string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.SetLevel(log.ErrorLevel)
	defer log.SetLevel(log.InfoLevel)

	ch := make(chan error)
	go func() {
		ch <- patchWithContext(timeoutCtx, ch, image, reportFile, patchedTag, workingFolder, scanner, format, output, ignoreError, bkOpts, out)
	}()

	select {
	case err := <-ch:
		if err == nil {
			return nil
		}
		return fmt.Errorf("copa: error patching image :: %w", err)
	case <-timeoutCtx.Done():
		// add a grace period for long running deferred cleanup functions to complete
		<-time.After(1 * time.Second)

		err := fmt.Errorf("patch exceeded timeout %v", timeout)
		log.Error(err)
		return err
	}
}

func removeIfNotDebug(workingFolder string) {
	if log.GetLevel() >= log.DebugLevel {
		// Keep the intermediate outputs for out1916a980ac0dputs solved to working folder if debugging
		log.Warnf("--debug specified, working folder at %s needs to be manually cleaned up", workingFolder)
	} else {
		os.RemoveAll(workingFolder)
	}
}

func patchWithContext(ctx context.Context, ch chan error, image, reportFile, patchedTag, workingFolder, scanner, format, output string, ignoreError bool, bkOpts buildkit.Opts, out string) error {
	imageName, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return err
	}
	if reference.IsNameOnly(imageName) {
		log.Warnf("Image name has no tag or digest, using latest as tag")
		imageName = reference.TagNameOnly(imageName)
	}
	taggedName, ok := imageName.(reference.Tagged)
	if !ok {
		err := errors.New("unexpected: TagNameOnly did not create Tagged ref")
		log.Error(err)
		return err
	}
	tag := taggedName.Tag()
	if patchedTag == "" {
		if tag == "" {
			log.Warnf("No output tag specified for digest-referenced image, defaulting to `%s`", defaultPatchedTagSuffix)
			patchedTag = defaultPatchedTagSuffix
		} else {
			patchedTag = fmt.Sprintf("%s-%s", tag, defaultPatchedTagSuffix)
		}
	}
	_, err = reference.WithTag(imageName, patchedTag)
	if err != nil {
		return fmt.Errorf("%w with patched tag %s", err, patchedTag)
	}
	patchedImageName := fmt.Sprintf("%s:%s", imageName.Name(), patchedTag)

	// Ensure working folder exists for call to InstallUpdates
	if workingFolder == "" {
		var err error
		workingFolder, err = os.MkdirTemp("", "copa-*")
		if err != nil {
			return err
		}
		defer removeIfNotDebug(workingFolder)
		if err = os.Chmod(workingFolder, 0o744); err != nil {
			return err
		}
	} else {
		if isNew, err := utils.EnsurePath(workingFolder, 0o744); err != nil {
			log.Errorf("failed to create workingFolder %s", workingFolder)
			return err
		} else if isNew {
			defer removeIfNotDebug(workingFolder)
		}
	}

	var updates *unversioned.UpdateManifest
	// Parse report for update packages
	if reportFile != "" {
		updates, err = report.TryParseScanReport(reportFile, scanner)
		if err != nil {
			return fmt.Errorf("copa: error parsing scan report %s :: %w", reportFile, err)
		}
		log.Debugf("updates to apply: %v", updates)
	}

	bkClient, err := buildkit.NewClient(ctx, bkOpts)
	if err != nil {
		return fmt.Errorf("copa: error creating buildkit client :: %w", err)
	}
	defer bkClient.Close()

	pipeR, pipeW := io.Pipe()
	dockerConfig := config.LoadDefaultConfigFile(os.Stderr)
	attachable := []session.Attachable{authprovider.NewDockerAuthProvider(dockerConfig, nil)}
	solveOpt := client.SolveOpt{
		Exports: []client.ExportEntry{
			{
				Type: client.ExporterOCI,
				Attrs: map[string]string{
					"name": patchedImageName,
				},
				Output: func(_ map[string]string) (io.WriteCloser, error) {
					return pipeW, nil
				},
			},
		},
		Frontend: "",         // i.e. we are passing in the llb.Definition directly
		Session:  attachable, // used for authprovider, sshagentprovider and secretprovider
	}
	solveOpt.SourcePolicy, err = build.ReadSourcePolicy()
	if err != nil {
		return fmt.Errorf("copa: error reading source policy :: %w", err)
	}

	buildChannel := make(chan *client.SolveStatus)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		_, err := bkClient.Build(ctx, solveOpt, copaProduct, func(ctx context.Context, c gwclient.Client) (*gwclient.Result, error) {
			// Configure buildctl/client for use by package manager
			config, err := buildkit.InitializeBuildkitConfig(ctx, c, imageName.String())
			if err != nil {
				ch <- err
				return nil, fmt.Errorf("copa: error initializing buildkit config for image %s :: %w", imageName.String(), err)
			}

			// Create package manager helper
			var manager pkgmgr.PackageManager
			if reportFile == "" {
				// determine OS family
				fileBytes, err := buildkit.ExtractFileFromState(ctx, c, &config.ImageState, "/etc/os-release")
				if err != nil {
					ch <- err
					return nil, fmt.Errorf("unable to extract /etc/os-release file from state %w", err)
				}

				osType, err := getOSType(ctx, fileBytes)
				if err != nil {
					ch <- err
					return nil, fmt.Errorf("copa: error getting os type :: %w", err)
				}

				osVersion, err := getOSVersion(ctx, fileBytes)
				if err != nil {
					ch <- err
					return nil, fmt.Errorf("copa: error getting os version :: %w", err)
				}

				// get package manager based on os family type
				manager, err = pkgmgr.GetPackageManager(osType, osVersion, config, workingFolder)
				if err != nil {
					ch <- err
					return nil, fmt.Errorf("copa: error getting package manager for ostype=%s, version=%s :: %w", osType, osVersion, err)
				}
				// do not specify updates, will update all
				updates = nil
			} else {
				// get package manager based on os family type
				manager, err = pkgmgr.GetPackageManager(updates.Metadata.OS.Type, updates.Metadata.OS.Version, config, workingFolder)
				if err != nil {
					ch <- err
					return nil, fmt.Errorf("copa: error getting package manager by family type: ostype=%s, osversion=%s :: %w", updates.Metadata.OS.Type, updates.Metadata.OS.Version, err)
				}
			}

			// Export the patched image state to Docker
			// TODO: Add support for other output modes as buildctl does.
			log.Infof("Patching %d vulnerabilities", len(updates.Updates))
			patchedImageState, errPkgs, err := manager.InstallUpdates(ctx, updates, ignoreError)
			log.Infof("Error is: %v", err)
			if err != nil {
				ch <- err
				return nil, nil
			}

			platform := platforms.Normalize(platforms.DefaultSpec())
			if platform.OS != "linux" {
				platform.OS = "linux"
			}

			def, err := patchedImageState.Marshal(ctx, llb.Platform(platform))
			if err != nil {
				ch <- err
				return nil, err
			}

			res, err := c.Solve(ctx, gwclient.SolveRequest{
				Definition: def.ToPB(),
				Evaluate:   true,
			})
			if err != nil {
				ch <- err
				return nil, err
			}

			res.AddMeta(exptypes.ExporterImageConfigKey, config.ConfigData)

			// Currently can only validate updates if updating via scanner
			if reportFile != "" {
				// create a new manifest with the successfully patched packages
				validatedManifest := &unversioned.UpdateManifest{
					Metadata: unversioned.Metadata{
						OS: unversioned.OS{
							Type:    updates.Metadata.OS.Type,
							Version: updates.Metadata.OS.Version,
						},
						Config: unversioned.Config{
							Arch: updates.Metadata.Config.Arch,
						},
					},
					Updates: []unversioned.UpdatePackage{},
				}
				for _, update := range updates.Updates {
					if !slices.Contains(errPkgs, update.Name) {
						validatedManifest.Updates = append(validatedManifest.Updates, update)
					}
				}
				// vex document must contain at least one statement
				if output != "" && len(validatedManifest.Updates) > 0 {
					if err := vex.TryOutputVexDocument(validatedManifest, manager, patchedImageName, format, output); err != nil {
						ch <- err
						return nil, err
					}
				}
			}

			return res, nil
		}, buildChannel)

		return err
	})

	eg.Go(func() error {
		// not using shared context to not disrupt display but let us finish reporting errors
		mode := progressui.AutoMode
		if log.GetLevel() >= log.DebugLevel {
			mode = progressui.PlainMode
		}
		display, err := progressui.NewDisplay(os.Stderr, mode)
		if err != nil {
			return err
		}

		_, err = display.UpdateFrom(ctx, buildChannel)
		return err
	})

	eg.Go(func() error {
		body, err := io.ReadAll(pipeR)
		if err != nil {
			return err
		}

		err = os.WriteFile(out, body, os.ModePerm)
		if err != nil {
			return err
		}

		return pipeR.Close()
	})

	return eg.Wait()
}

func getOSType(ctx context.Context, osreleaseBytes []byte) (string, error) {
	r := bytes.NewReader(osreleaseBytes)
	osData, err := osrelease.Parse(ctx, r)
	if err != nil {
		return "", fmt.Errorf("unable to parse os-release data %w", err)
	}

	osType := strings.ToLower(osData["NAME"])
	switch {
	case strings.Contains(osType, "alpine"):
		return "alpine", nil
	case strings.Contains(osType, "debian"):
		return "debian", nil
	case strings.Contains(osType, "ubuntu"):
		return "ubuntu", nil
	case strings.Contains(osType, "amazon"):
		return "amazon", nil
	case strings.Contains(osType, "centos"):
		return "centos", nil
	case strings.Contains(osType, "mariner"):
		return "cbl-mariner", nil
	case strings.Contains(osType, "red hat"):
		return "redhat", nil
	default:
		log.Error("unsupported osType", osType)
		return "", errors.ErrUnsupported
	}
}

func getOSVersion(ctx context.Context, osreleaseBytes []byte) (string, error) {
	r := bytes.NewReader(osreleaseBytes)
	osData, err := osrelease.Parse(ctx, r)
	if err != nil {
		return "", fmt.Errorf("unable to parse os-release data %w", err)
	}

	return osData["VERSION_ID"], nil
}
