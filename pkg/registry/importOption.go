package registry

import (
	"context"
	"log/slog"

	"github.com/ChristofferNissen/helmper/pkg/image"

	"github.com/ChristofferNissen/helmper/pkg/util/bar"
	"golang.org/x/sync/errgroup"
)

type ImportOption struct {
	Data map[*Registry]map[*image.Image]bool

	Architecture *string
	All          bool
}

func (io ImportOption) Run(ctx context.Context) error {

	slog.Debug("pushing images to registries..")

	size := func() int {
		size := 0
		for _, m := range io.Data {
			for _, b := range m {
				if b {
					size++
				}
			}
		}
		return size
	}()

	bar := bar.New("Pushing images...\r", size)

	eg, egCtx := errgroup.WithContext(ctx)
	for r, m := range io.Data {
		for i, b := range m {
			eg.Go(func() error {
				if io.All || b {
					name, err := i.ImageName()
					if err != nil {
						return err
					}
					manifest, err := r.Push(egCtx, i.Registry, name, i.Tag, io.Architecture)
					if err != nil {
						return err
					}
					i.Digest = manifest.Digest.String()
					_ = bar.Add(1)
				}
				return nil
			})
		}
	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	_ = bar.Finish()
	slog.Debug("all images have been pushed to registries")
	return nil
}
