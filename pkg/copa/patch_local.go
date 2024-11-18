package copa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/containerd/platforms"
	"github.com/docker/buildx/build"
	"github.com/docker/cli/cli/config"
	"github.com/google/go-containerregistry/pkg/crane"
	log "github.com/sirupsen/logrus"
	"github.com/tonistiigi/fsutil"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"

	"github.com/distribution/reference"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/client/llb/sourceresolver"
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

// Patch command applies package updates to an OCI image given a vulnerability report.
func PatchLocal(ctx context.Context, timeout time.Duration, image, path, reportFile, patchedTag, workingFolder, scanner, format, output string, ignoreError bool, bkOpts buildkit.Opts, out string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.SetLevel(log.ErrorLevel)
	defer log.SetLevel(log.InfoLevel)

	ch := make(chan error)
	go func() {
		ch <- patchLocalWithContext(timeoutCtx, ch, image, path, reportFile, patchedTag, workingFolder, scanner, format, output, ignoreError, bkOpts, out)
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

func patchLocalWithContext(ctx context.Context, ch chan error, image, path, reportFile, patchedTag, workingFolder, scanner, format, output string, ignoreError bool, bkOpts buildkit.Opts, out string) error {
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

	// store, err := oci.NewFromTar(ctx, path)
	// if err != nil {
	// 	return err
	// }
	tempDir, err := os.MkdirTemp("", "oras_oci_example_*")
	if err != nil {
		panic(err) // Handle error
	}
	defer os.RemoveAll(tempDir)
	// target, err := oci.New(tempDir)
	// if err != nil {
	// 	panic(err) // Handle error
	// }
	// if _, err := oras.Copy(ctx, store, tag, target, tag, oras.DefaultCopyOptions); err != nil {
	// 	return err
	// }

	// TODO: Fix this when i figure out how to convert from oci to docker
	v1, err := crane.Pull(image)
	if err != nil {
		panic(err) // Handle error
	}
	err = crane.SaveLegacy(v1, tag, tempDir+"/image.tar")
	if err != nil {
		panic(err) // Handle error
	}

	// Extract the tar file
	cmd := exec.Command("tar", "xf", tempDir+"/image.tar", "-C", tempDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract tar: %w", err)
	}

	// Read manifest.json to get the config filename
	manifestData, err := os.ReadFile(tempDir + "/manifest.json")
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}
	var manifests []map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifests); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}
	if len(manifests) == 0 {
		return fmt.Errorf("empty manifest file")
	}

	// Get the first layer file name from the manifest
	layers := manifests[0]["Layers"].([]interface{})
	if len(layers) == 0 {
		return fmt.Errorf("no layers found in manifest")
	}
	layerTarGz := layers[0].(string)

	// Extract the layer tar.gz
	cmd = exec.Command("tar", "xvzf", tempDir+"/"+layerTarGz, "-C", tempDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract layer: %w", err)
	}

	// cmd := exec.Command("tar", "xf", tempDir+"/image.tar", "-C", tempDir)
	// if err := cmd.Run(); err != nil {
	// 	return fmt.Errorf("failed to extract tar: %w", err)
	// }

	// Remove the tar file
	if err := os.Remove(tempDir + "/image.tar"); err != nil {
		return fmt.Errorf("failed to remove tar file: %w", err)
	}

	// img, err := tarball.ImageFromPath(path, nil)
	// if err != nil {
	// 	return err
	// }
	// log.Println(img)

	fs, _ := fsutil.NewFS(tempDir)

	pipeR, pipeW := io.Pipe()
	dockerConfig := config.LoadDefaultConfigFile(os.Stderr)
	attachable := []session.Attachable{authprovider.NewDockerAuthProvider(dockerConfig, nil)}
	solveOpt := client.SolveOpt{
		LocalMounts: map[string]fsutil.FS{
			tempDir: fs,
		},
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
			config, err := InitializeBuildkitConfig(ctx, c, imageName.String(), tempDir)
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

func InitializeBuildkitConfig(ctx context.Context, c gwclient.Client, image, path string) (*buildkit.Config, error) {

	// Initialize buildkit config for the target image
	config := buildkit.Config{
		ImageName: image,
	}

	// // Create an llb.State from a local file
	// localFileState := llb.Local("local-src",
	// 	llb.FollowPaths([]string{"path/to/local/file"}),
	// 	llb.IncludePatterns([]string{"file"}),
	// 	llb.SessionID(c.BuildOpts().SessionID),
	// 	llb.SharedKeyHint("local-file"),
	// 	llb.WithCustomName("[internal] load local file"),
	// )

	// Add the local file state to the buildkit config
	// config.ImageState = localFileState

	// Extract all strings from the path, which will be the full path to a tar or oci-layout image on filesystem
	// tarFileState := llb.Local("local-taroci",
	// 	llb.FollowPaths([]string{path}),
	// 	llb.IncludePatterns([]string{"*.tar", "oci-layout"}),
	// 	llb.SessionID(c.BuildOpts().SessionID),
	// 	llb.SharedKeyHint("local-tar"),
	// 	llb.WithCustomName("[internal] load local tar"),
	// )

	// Resolve the image config for the target image
	_, _, configData, err := c.ResolveImageConfig(ctx, image, sourceresolver.Opt{
		ImageOpt: &sourceresolver.ResolveImageOpt{
			ResolveMode: llb.ResolveModePreferLocal.String(),
		},
	})
	if err != nil {
		return nil, err
	}

	config.ConfigData = configData

	// tarFileState, err := llb.OCILayout(path).WithImageConfig(config.ConfigData)
	// if err != nil {
	// 	return nil, err
	// }

	tarFileState, err := llb.Local(path,
		llb.IncludePatterns([]string{"*"}),
		llb.SessionID(c.BuildOpts().SessionID),
		llb.SharedKeyHint("local-tar"),
		llb.WithCustomName("[internal] load local tar"),
	).WithImageConfig(config.ConfigData)
	if err != nil {
		return nil, err
	}

	// Add the tar file state to the buildkit config
	config.ImageState = tarFileState

	// Load the target image state with the resolved image config in case environment variable settings
	// are necessary for running apps in the target image for updates
	// config.ImageState, err = llb.Image(image,
	// 	llb.ResolveModePreferLocal,
	// 	llb.WithMetaResolver(c),
	// ).WithImageConfig(config.ConfigData)
	// if err != nil {
	// 	return nil, err
	// }

	config.Client = c

	return &config, nil
}
