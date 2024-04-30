package output

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/ChristofferNissen/helmper/helmper/internal/bootstrap"
	"github.com/ChristofferNissen/helmper/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/helmper/pkg/util/counter"
	"github.com/ChristofferNissen/helmper/helmper/pkg/util/file"
	"github.com/ChristofferNissen/helmper/helmper/pkg/util/state"
	"github.com/ChristofferNissen/helmper/helmper/pkg/util/terminal"
	"github.com/jedib0t/go-pretty/table"
)

var sc counter.SafeCounter = counter.NewSafeCounter()

// create a new table.writer with header and os.Stdout output mirror
func newTable(row table.Row) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(row)
	return t
}

func renderChartTable(rows []table.Row) {
	t := newTable(table.Row{"#", "Type", "Chart", "Version", "Latest Version", "Latest", "Values", "SubChart", "Version", "Condition", "Enabled"})
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
		latest, _ := c.LatestVersion()

		_, chartRef, values, _ := c.Read(args.Update)
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
	t := newTable(table.Row{"#", "Helm Chart", "Chart Version", "Helm Value Path", "Image"})
	id := 0
	for c, v := range chartImageHelmValuesMap {
		for i, paths := range v {
			ref, _ := i.String()
			noSHA := strings.SplitN(ref, "@", 2)[0]
			t.AppendRow(table.Row{id, c.Name, c.Version, strings.Join(paths, "\n"), noSHA})
			id = id + 1
		}
	}
	t.Render()
}

func getImportTableRow(_ context.Context, viper *viper.Viper, c helm.Chart, image string, keys []string, m map[string]bool) table.Row {
	importImage := true

	row := table.Row{}
	row = append(row, sc.Value("index_import"), c.Name, c.Version, image)

	for _, key := range keys {
		if key != "prd" {
			row = append(row, terminal.StatusEmoji(m[key]))
			importImage = importImage && !m[key]
		}
	}

	ic := state.GetValue[bootstrap.ImportConfigSection](viper, "importConfig")
	if ic.Import.Enabled {
		row = append(row, terminal.StatusEmoji(state.GetValue[bool](viper, "all") || importImage))
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

		var seenImages []registry.Image = make([]registry.Image, 0)
		for i := range m {
			if !i.In(seenImages) {
				// make sure we don't parse again
				seenImages = append(seenImages, *i)

				// check if image exists in registry
				m, _ := registry.Exists(ctx, i, registries)

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

	// dynamic number of registries
	for _, r := range registries {
		name := r.GetName()
		header = append(header, name)
		footer = append(footer, "")
	}

	ic := state.GetValue[bootstrap.ImportConfigSection](viper, "importConfig")
	if ic.Import.Enabled {
		// second static part of header
		header = append(header, "import")
		footer = append(footer, missing)
	}

	// construct table
	t := newTable(header)
	t.AppendRows(rows)
	t.AppendFooter(footer)
	t.Render()
	return nil
}
