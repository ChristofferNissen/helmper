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
	"helm.sh/helm/v3/pkg/repo"
)

type ChartImportOption struct {
	Registries      []registry.Registry
	ChartCollection *ChartCollection
	All             bool
	ModifyRegistry  bool
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

	charts := []Chart{}
	for _, c := range opt.ChartCollection.Charts {

		_, chartRef, _, err := c.Read(args.Update)
		if err != nil {
			return err
		}

		c.DepsCount = len(chartRef.Metadata.Dependencies)
		charts = append(charts, c)

		for _, d := range chartRef.Metadata.Dependencies {

			// We need all dependencies for the chart to be available in the registry to do 'helm dpt up'
			// if !ConditionMet(d.Condition, values) {
			// 	slog.Debug("Skipping disabled chart", slog.String("chart", d.Name), slog.String("condition", d.Condition))
			// 	continue
			// }

			// Only import enabled charts
			if d.Repository == "" {
				// Embedded in parent chart
				slog.Debug("Skipping embedded chart", slog.String("chart", d.Name), slog.String("parent", c.Name))
				continue
			}

			chart := Chart{
				Name: d.Name,
				Repo: repo.Entry{
					Name: c.Repo.Name + "/" + d.Name,
					URL:  d.Repository,
				},
				Version:        d.Version,
				ValuesFilePath: c.ValuesFilePath,
				Parent:         &c,
				DepsCount:      0,
				PlainHTTP:      c.PlainHTTP,
			}

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

	// Sort charts according to least dependencies
	sort.Slice(charts, func(i, j int) bool { return charts[i].DepsCount < charts[j].DepsCount })

	bar := progressbar.NewOptions(len(charts),
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

	for _, c := range charts {

		if c.Name == "images" {
			continue
		}

		for _, r := range opt.Registries {
			registryURL := "oci://" + r.URL + "/charts"
			if !opt.All {
				_, err := r.Exist(ctx, "charts/"+c.Name, c.Version)
				if err == nil {
					slog.Info("Chart already present in registry. Skipping import", slog.String("chart", "charts/"+c.Name), slog.String("registry", "oci://"+r.URL), slog.String("version", c.Version))
					continue
				}
				slog.Debug(err.Error())
			}

			if opt.ModifyRegistry {
				res, err := c.PushAndModify(registryURL, r.Insecure, r.PlainHTTP)
				if err != nil {
					return fmt.Errorf("helm: error pushing and modifying chart %s to registry %s :: %w", c.Name, registryURL, err)
				}
				slog.Debug(res)

				continue
			}

			res, err := c.Push(registryURL, r.Insecure, r.PlainHTTP)
			if err != nil {
				return fmt.Errorf("helm: error pushing chart %s to registry %s :: %w", c.Name, registryURL, err)
			}
			slog.Debug(res)

		}

		_ = bar.Add(1)
	}

	return bar.Finish()

}
