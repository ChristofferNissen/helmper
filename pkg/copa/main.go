package copa

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os/exec"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	v1_spec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	myBar "github.com/ChristofferNissen/helmper/pkg/util/bar"
)

func SupportedOS(os string) bool {
	switch os {
	case "photon":
		return false
	default:
		return true
	}
}

type PatchOption struct {
	Data map[*registry.Registry]map[*image.Image]bool

	TarFolder    string
	ReportFolder string

	Buildkit struct {
		Addr       string
		CACertPath string
		CertPath   string
		KeyPath    string
	}

	IgnoreErrors bool
	Architecture *string
}

func (o PatchOption) Run(ctx context.Context, reportFilePaths map[*image.Image]string, outFilePaths map[*image.Image]string) error {
	size := func() int {
		size := 0
		for _, m := range o.Data {
			for _, b := range m {
				if b {
					size++
				}
			}
		}
		return size
	}()

	if size <= 0 {
		return nil
	}

	bar := myBar.New("Patching images...\r", size)

	seenImages := []image.Image{}
	for _, m := range o.Data {
		for i := range m {
			ref := i.String()

			if i.In(seenImages) {
				log.Printf("Already patched '%s', skipping...\n", ref)
				continue
			}
			// make sure we don't parse again
			seenImages = append(seenImages, *i)

			cmdArgs := []string{"patch", "--timeout", "30m", "--image", ref, "--report", reportFilePaths[i], "--tag", i.Tag}
			// TODO: Figure out if we need to remove this and associated config in viper
			// if o.IgnoreErrors {
			// 	cmdArgs = append(cmdArgs, "--ignore-errors")
			// }
			if o.Buildkit.Addr != "" {
				cmdArgs = append(cmdArgs, "--addr", o.Buildkit.Addr)
			}
			if o.Buildkit.CACertPath != "" {
				cmdArgs = append(cmdArgs, "--cacert", o.Buildkit.CACertPath)
			}
			if o.Buildkit.CertPath != "" {
				cmdArgs = append(cmdArgs, "--cert", o.Buildkit.CertPath)
			}
			if o.Buildkit.KeyPath != "" {
				cmdArgs = append(cmdArgs, "--key", o.Buildkit.KeyPath)
			}

			cmd := exec.CommandContext(ctx, "copa", cmdArgs...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("error patching image %s :: %w :: %s", ref, err, string(output))
			}

			// Save the patched image to a tar file
			cmd = exec.CommandContext(ctx, "docker", "save", "-o", outFilePaths[i], ref)
			output, err = cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("error saving image %s to tar :: %w :: %s", i.Tag, err, string(output))
			}

			_ = bar.Add(1)
		}
	}
	_ = bar.Finish()

	bar = myBar.New("Pushing images from tar...\r", size)
	for r, m := range o.Data {
		for i, b := range m {
			if b {
				name, _ := i.ImageName()

				store, err := oci.NewFromTar(ctx, outFilePaths[i])
				if err != nil {
					return err
				}
				manifest, err := store.Resolve(ctx, i.Tag)
				if err != nil {
					return err
				}
				i.Digest = manifest.Digest.String()

				if r.PrefixSource {
					old := name
					name, _ = image.UpdateNameWithPrefixSource(i)
					slog.Info("registry has PrefixSource enabled", slog.String("old", old), slog.String("new", name))
				}

				// Connect to a remote repository
				url, _ := strings.CutPrefix(r.URL, "oci://")
				repo, err := remote.NewRepository(url + "/" + name)
				if err != nil {
					return err
				}

				repo.PlainHTTP = r.PlainHTTP

				// Prepare authentication using Docker credentials
				storeOpts := credentials.StoreOptions{}
				credStore, err := credentials.NewStoreFromDocker(storeOpts)
				if err != nil {
					return err
				}
				repo.Client = &auth.Client{
					Client:     retry.DefaultClient,
					Cache:      auth.NewCache(),
					Credential: credentials.Credential(credStore), // Use the credentials store
				}

				// Copy from the file store to the remote repository
				opts := oras.DefaultCopyOptions
				if o.Architecture != nil {
					v, err := v1.ParsePlatform(*o.Architecture)
					if err != nil {
						return err
					}
					opts.WithTargetPlatform(
						&v1_spec.Platform{
							Architecture: v.Architecture,
							OS:           v.OS,
							OSVersion:    v.OSVersion,
							OSFeatures:   v.OSFeatures,
							Variant:      v.Variant,
						},
					)
				}
				manifest, err = oras.Copy(ctx, store, i.Tag, repo, i.Tag, opts)
				if err != nil {
					return err
				}

				i.Digest = manifest.Digest.String()
			}
			_ = bar.Add(1)
		}
	}

	_ = bar.Finish()

	return nil
}
