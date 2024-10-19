package helm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

type ChartImportOption struct {
	Data           map[*registry.Registry]map[*Chart]bool
	All            bool
	ModifyRegistry bool
}

func (opt ChartImportOption) Run(ctx context.Context, setters ...Option) error {

	// Default Options
	args := &Options{
		Verbose:    false,
		Update:     false,
		K8SVersion: "1.27.16",
	}

	for _, setter := range setters {
		setter(args)
	}

	size := func() int {
		size := 0
		for _, m := range opt.Data {
			for _, b := range m {
				if b {
					size++
				}
			}
		}
		return size
	}()

	if !(size > 0) {
		return nil
	}

	bar := progressbar.NewOptions(size,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()), // "github.com/k0kubun/go-ansi"
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionSetDescription("Pushing charts...\r"),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	for r, m := range opt.Data {
		charts := []Chart{}

		for c, b := range m {
			if b {

				_, chartRef, _, err := c.Read(args.Update)
				if err != nil {
					return err
				}

				c.DepsCount = len(chartRef.Metadata.Dependencies)
				charts = append(charts, *c)

				for _, d := range chartRef.Metadata.Dependencies {

					// We need all dependencies for the chart to be available in the registry to do 'helm dpt up'
					// if !ConditionMet(d.Condition, values) {
					// 	slog.Debug("Skipping disabled chart", slog.String("chart", d.Name), slog.String("condition", d.Condition))
					// 	continue
					// }

					// only import remote charts
					if d.Repository == "" || strings.HasPrefix(d.Repository, "file://") {
						// Embedded in parent chart
						slog.Debug("Skipping embedded chart", slog.String("chart", d.Name), slog.String("parent", c.Name))
						continue
					}

					chart := DependencyToChart(d, *c)

					// Resolve Globs to latest patch
					if strings.Contains(chart.Version, "*") {
						v, err := chart.ResolveVersion()
						if err == nil {
							chart.Version = v
						}
					}

					charts = append(charts, chart)
				}
			}
		}

		// Sort charts according to least dependencies
		sort.Slice(charts, func(i, j int) bool {
			return charts[i].DepsCount < charts[j].DepsCount
		})

		for _, c := range charts {

			if c.Name == "images" {
				continue
			}

			if !opt.All {
				_, err := r.Exist(ctx, "charts/"+c.Name, c.Version)
				if err == nil {
					slog.Info("Chart alreregistryURLady present in registry. Skipping import", slog.String("chart", "charts/"+c.Name), slog.String("registry", "oci://"+r.URL), slog.String("version", c.Version))
					continue
				}
				slog.Debug(err.Error())
			}

			if opt.ModifyRegistry {
				res, err := c.PushAndModify(r.URL, r.Insecure, r.PlainHTTP)
				if err != nil {
					registryURL := "oci://" + r.URL + "/charts"
					return fmt.Errorf("helm: error pushing and modifying chart %s to registry %s :: %w", c.Name, registryURL, err)
				}
				slog.Debug(res)
				_ = bar.Add(1)
				continue
			}

			res, err := c.Push(r.URL, r.Insecure, r.PlainHTTP)
			if err != nil {
				registryURL := "oci://" + r.URL + "/charts"
				return fmt.Errorf("helm: error pushing chart %s to registry %s :: %w", c.Name, registryURL, err)
			}
			slog.Debug(res)

			_ = bar.Add(1)
		}
	}

	return bar.Finish()

}
