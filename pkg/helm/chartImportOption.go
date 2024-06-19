package helm

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"helm.sh/helm/v3/pkg/repo"
)

type ChartImportOption struct {
	Registries      []registry.Registry
	ChartCollection *ChartCollection
}

func (opt ChartImportOption) Run(ctx context.Context, setters ...Option) error {

	// Default Options
	args := &Options{
		Verbose:    false,
		Update:     false,
		K8SVersion: "1.27.7",
	}

	for _, setter := range setters {
		setter(args)
	}

	charts := []Chart{}
	for _, c := range opt.ChartCollection.Charts {
		charts = append(charts, c)

		_, chartRef, values, err := c.Read(args.Update)
		if err != nil {
			return err
		}

		for _, d := range chartRef.Metadata.Dependencies {

			if !ConditionMet(d.Condition, values) {
				slog.Debug("Skipping disabled chart", slog.String("chart", d.Name), slog.String("condition", d.Condition))
				continue
			}

			// Only import enabled charts
			if d.Repository == "" {
				// Embedded in parent chart
				slog.Debug("Skipping embedded chart", slog.String("chart", d.Name), slog.String("parent", c.Name))
				continue
			}

			chart := Chart{
				Name: d.Name,
				Repo: repo.Entry{
					Name: c.Repo.Name,
					URL:  d.Repository,
				},
				Version:        d.Version,
				ValuesFilePath: c.ValuesFilePath,
				Parent:         &c,
			}

			// Resolve Globs to latest patch
			v, err := chart.ResolveVersion()
			if err != nil {
				return err
			}
			chart.Version = v

			charts = append(charts, chart)
		}

	}

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

			_, err := r.Exist(ctx, "charts/"+c.Name, c.Version)
			if err == nil {
				slog.Info("Chart already present in registry. Skipping import", slog.String("chart", "charts/"+c.Name), slog.String("registry", "oci://"+r.URL))
				continue
			}

			slog.Debug(err.Error())

			res, err := c.Push("oci://"+r.URL+"/charts", r.Insecure, r.PlainHTTP)
			if err != nil {
				return err
			}

			slog.Debug(res)

		}

		_ = bar.Add(1)
	}

	return bar.Finish()

}
