package output

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/ChristofferNissen/helmper/internal/bootstrap"
	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/util/counter"
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
		keys = append(keys, r.URL)
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
		header = append(header, r.GetName())
		footer = append(footer, "")

		if ic.Import.Enabled {
			// second static part of header
			header = append(header, "import")
			footer = append(footer, sc.Value(r.URL))
		}
	}

	// construct table
	t := newTable("Registry Overview For Images", header)
	t.AppendRows(rows)
	t.AppendFooter(footer)
	t.Render()

	return nil
}

func RenderChartOverviewTable(ctx context.Context, viper *viper.Viper, missing int, registries []registry.Registry, charts helm.ChartCollection) error {

	// Create collection of registry names as keys for iterating registries
	keys := make([]string, 0)
	for _, r := range registries {
		keys = append(keys, r.URL)
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
		header = append(header, r.GetName())
		footer = append(footer, "")

		if ic.Import.Enabled {
			// second static part of header
			header = append(header, "import")
			footer = append(footer, sc.Value(r.URL+"charts"))
		}
	}

	// construct table
	t := newTable("Registry Overview For Charts", header)
	t.AppendRows(rows)
	t.AppendFooter(footer)
	t.Render()

	return nil
}
