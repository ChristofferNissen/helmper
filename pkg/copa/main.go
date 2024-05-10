package copa

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/k0kubun/go-ansi"
	"github.com/project-copacetic/copacetic/pkg/buildkit"
	"github.com/schollz/progressbar/v3"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type PatchOption struct {
	Imgs       []*registry.Image
	Registries []registry.Registry

	TarFolder    string
	ReportFolder string

	Buildkit struct {
		Addr       string
		CACertPath string
		CertPath   string
		KeyPath    string
	}

	IgnoreErrors bool
}

func (o PatchOption) Run(ctx context.Context, reportFilePaths map[*registry.Image]string, outFilePaths map[*registry.Image]string) error {

	bar := progressbar.NewOptions(len(o.Imgs),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()), // "github.com/k0kubun/go-ansi"
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetDescription("Patching images...\r"),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	for _, i := range o.Imgs {
		ref, _ := i.String()

		if err := Patch(ctx, 30*time.Minute, ref, reportFilePaths[i], i.Tag, "", "trivy", "openvex", "", o.IgnoreErrors, buildkit.Opts{
			Addr:       o.Buildkit.Addr,
			CACertPath: o.Buildkit.CACertPath,
			CertPath:   o.Buildkit.CertPath,
			KeyPath:    o.Buildkit.KeyPath,
		}, outFilePaths[i]); err != nil {
			return err
		}

		_ = bar.Add(1)
	}

	_ = bar.Finish()

	bar = progressbar.NewOptions(len(o.Imgs),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()), // "github.com/k0kubun/go-ansi"
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetDescription("Pushing images from tar...\r"),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	for _, i := range o.Imgs {
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

		for _, r := range o.Registries {
			// Connect to a remote repository
			repo, err := remote.NewRepository(r.URL + "/" + name)
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
			manifest, err = oras.Copy(ctx, store, i.Tag, repo, i.Tag, oras.DefaultCopyOptions)
			if err != nil {
				return err
			}

			i.Digest = manifest.Digest.String()

		}

		_ = bar.Add(1)
	}

	_ = bar.Finish()

	return nil
}
