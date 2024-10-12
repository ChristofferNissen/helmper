package output

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/ChristofferNissen/helmper/internal/bootstrap"
	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/util/counter"
	"github.com/ChristofferNissen/helmper/pkg/util/file"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
	"github.com/ChristofferNissen/helmper/pkg/util/terminal"
	"github.com/jedib0t/go-pretty/v6/table"
)

var sc counter.SafeCounter = counter.NewSafeCounter()

// create a new table.writer with header and os.Stdout output mirror
func newTable(title string, header table.Row) table.Writer {
	t := table.NewWriter()
	t.SetTitle(title)
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(header)
	return t
}

func renderChartTable(rows []table.Row) {
	t := newTable("Charts", table.Row{"#", "Type", "Chart", "Version", "Latest Version", "Latest", "Values", "SubChart", "Version", "Condition", "Enabled"})
	t.AppendRows(rows)
	t.SortBy([]table.SortBy{
		{Number: 1, Mode: table.AscNumeric},
	})
	t.Render()
}

func determinePathType(path string) string {
	// Output Table
	if file.Exists(path) {
		return "custom"
	}
	return "default"
}

func RenderChartTable(charts *helm.ChartCollection, setters ...Option) {

	// Default Options
	args := &Options{
		Update: false,
	}

	for _, setter := range setters {
		setter(args)
	}

	var sc counter.SafeCounter = counter.NewSafeCounter()

	var rows []table.Row = make([]table.Row, 0)
	for _, c := range charts.Charts {

		// Check for latest version of chart
		latest, err := c.LatestVersion()
		if err != nil {
			slog.Error(err.Error(), slog.String("chart", c.Name), slog.String("repo", c.Repo.URL), slog.String("version", c.Version))
		}

		_, chartRef, values, err := c.Read(args.Update)
		if err != nil {
			slog.Error(err.Error(), slog.String("chart", c.Name), slog.String("version", c.Version))
			continue
		}
		valuesType := determinePathType(c.ValuesFilePath)

		rows = append(rows,
			table.Row{sc.Value("charts"), "Chart", c.Name, c.Version, latest, terminal.StatusEmoji(c.Version == latest), valuesType, "", "", "", ""},
		)

		// reserve ids for table output
		sc.Inc("charts")
		count := len(chartRef.Metadata.Dependencies)
		reservedIDs := make([]int, count)
		for i := 0; i < count; i++ {
			reservedIDs[i] = sc.Value("charts")
			sc.Inc("charts")
		}

		// Look at SubCharts if they are enabled (chart dependency condition satisfied in values.yaml)
		for id, d := range chartRef.Metadata.Dependencies {

			// subchart enabled in main chart?
			enabled := helm.ConditionMet(d.Condition, values)
			slog.Debug(
				"SubChart enabled by condition in parent chart",
				slog.String("subChartName", d.Name),
				slog.String("parentChartName", chartRef.Name()),
				slog.String("condition", d.Condition),
				slog.Bool("enabled", enabled))

			// output table
			rows = append(rows,
				table.Row{reservedIDs[id], "Subchart", "", "", "", "", "parent", d.Name, d.Version, d.Condition, terminal.StatusEmoji(enabled)},
			)
		}
	}

	renderChartTable(rows)
}

func RenderHelmValuePathToImageTable(chartImageHelmValuesMap map[helm.Chart]map[*registry.Image][]string) {
	// Print Helm values to be set for each chart
	t := newTable("Helm Values Paths Per Image", table.Row{"#", "Helm Chart", "Chart Version", "Image", "Helm Value Path(s)"})
	id := 0
	for c, v := range chartImageHelmValuesMap {
		for i, paths := range v {
			ref, _ := i.String()
			noSHA := strings.SplitN(ref, "@", 2)[0]
			t.AppendRow(table.Row{id, c.Name, c.Version, noSHA, strings.Join(paths, "\n")})
			id = id + 1
		}
	}
	t.Render()
}

func getImportTableRow(_ context.Context, viper *viper.Viper, c helm.Chart, image string, keys []string, m map[string]bool) table.Row {
	row := table.Row{}
	row = append(row, sc.Value("index_import"), c.Name, c.Version, image)

	for _, key := range keys {
		row = append(row, terminal.StatusEmoji(m[key]))

		ic := state.GetValue[bootstrap.ImportConfigSection](viper, "importConfig")
		if ic.Import.Enabled {
			b := state.GetValue[bool](viper, "all") || !m[key]
			if b {
				sc.Inc(key)
			}
			row = append(row, terminal.StatusEmoji(b))
		}
	}

	sc.Inc("index_import")
	return row
}

func getImportTableRows(ctx context.Context, viper *viper.Viper, registries []registry.Registry, chartImageValuesMap map[helm.Chart]map[*registry.Image][]string) ([]table.Row, error) {

	// Create collection of registry names as keys for iterating registries
	keys := make([]string, 0)
	for _, r := range registries {
		keys = append(keys, r.GetName())
	}

	// Combine results
	rows := make([]table.Row, 0)

	for c, m := range chartImageValuesMap {
		seenImages := make([]registry.Image, 0)
		for i := range m {
			if !i.In(seenImages) {
				// make sure we don't parse again
				seenImages = append(seenImages, *i)

				name, err := i.ImageName()
				if err != nil {
					return []table.Row{}, err
				}
				// check if image exists in registry
				m := registry.Exists(ctx, name, i.Tag, registries)

				// add row to overview table
				ref, _ := i.String()
				row := getImportTableRow(ctx, viper, c, ref, keys, m)
				rows = append(rows, row)
			}
		}
	}

	return rows, nil
}

func RenderImageOverviewTable(ctx context.Context, viper *viper.Viper, missing int, registries []registry.Registry, chartImageValuesMap map[helm.Chart]map[*registry.Image][]string) error {

	rows, err := getImportTableRows(ctx, viper, registries, chartImageValuesMap)
	if err != nil {
		return err
	}

	header := table.Row{}
	footer := table.Row{}

	// first static part of header
	header = append(header, "#", "Helm Chart", "Chart Version", "Image")
	footer = append(footer, "", "", "", "")

	ic := state.GetValue[bootstrap.ImportConfigSection](viper, "importConfig")

	// dynamic number of registries
	for _, r := range registries {
		name := r.GetName()
		header = append(header, name)
		footer = append(footer, "")

		if ic.Import.Enabled {
			// second static part of header
			header = append(header, "import")
			footer = append(footer, sc.Value(name))
		}
	}

	// construct tab"test"le
	t := newTable("Registry Import Overview For Images", header)
	t.AppendRows(rows)
	t.AppendFooter(footer)
	t.Render()

	return nil
}

func RenderChartOverviewTable(ctx context.Context, viper *viper.Viper, missing int, registries []registry.Registry, charts helm.ChartCollection) error {

	// Create collection of registry names as keys for iterating registries
	keys := make([]string, 0)
	for _, r := range registries {
		keys = append(keys, r.GetName())
	}

	// Combine results
	rows := make([]table.Row, 0)
	for _, c := range charts.Charts {
		// check if image exists in registry
		m := registry.Exists(ctx, fmt.Sprintf("charts/%s", c.Name), c.Version, registries)

		// add row to overview table
		row := func() table.Row {
			row := table.Row{}
			row = append(row, sc.Value("index_import_charts"), c.Name, c.Version)

			for _, key := range keys {
				row = append(row, terminal.StatusEmoji(m[key]))
				ic := state.GetValue[bootstrap.ImportConfigSection](viper, "importConfig")
				if ic.Import.Enabled {
					b := state.GetValue[bool](viper, "all") || !m[key]
					if b {
						sc.Inc(key + "charts")
					}
					row = append(row, terminal.StatusEmoji(b))
				}
			}

			sc.Inc("index_import_charts")
			return row
		}()

		rows = append(rows, row)
	}

	header := table.Row{}
	footer := table.Row{}

	// first static part of header
	header = append(header, "#", "Helm Chart", "Chart Version")
	footer = append(footer, "", "", "")

	ic := state.GetValue[bootstrap.ImportConfigSection](viper, "importConfig")

	// dynamic number of registries
	for _, r := range registries {
		name := r.GetName()
		header = append(header, name)
		footer = append(footer, "")

		if ic.Import.Enabled {
			// second static part of header
			header = append(header, "import")
			footer = append(footer, sc.Value(name+"charts"))
		}
	}

	// construct tab"test"le
	t := newTable("Registry Import Overview For Images", header)
	t.AppendRows(rows)
	t.AppendFooter(footer)
	t.Render()

	return nil
}
