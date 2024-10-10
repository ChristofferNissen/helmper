package registry

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
)

type ImportOption struct {
	Imgs       []*Image
	Registries []Registry

	Architecture *string
	All          bool
}

func (io ImportOption) Run(ctx context.Context) error {

	slog.Debug("pushing images to registries..")

	bar := progressbar.NewOptions(len(io.Imgs), progressbar.OptionSetWriter(ansi.NewAnsiStdout()), // "github.com/k0kubun/go-ansi"
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("Pushing images...\r"),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	eg, egCtx := errgroup.WithContext(ctx)
	for _, i := range io.Imgs {
		status := Exists(ctx, i, io.Registries)

		func(i *Image) {
			eg.Go(func() error {
				for _, reg := range io.Registries {
					if io.All || !status[reg.GetName()] {
						name, err := i.ImageName()
						if err != nil {
							return err
						}
						manifest, err := reg.Push(egCtx, i.Registry, name, i.Tag, io.Architecture)
						if err != nil {
							return err
						}
						i.Digest = manifest.Digest.String()
					}
				}

				_ = bar.Add(1)

				return nil
			})
		}(i)

	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	_ = bar.Finish()

	slog.Debug("all images have been pushed to registries")

	return nil
}
