package helm

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/ChristofferNissen/helmper/pkg/myTable"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/util/counter"
	"github.com/ChristofferNissen/helmper/pkg/util/terminal"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

type ChartImportOption struct {
	Data           RegistryChartStatus
	All            bool
	ModifyRegistry bool
}

// Converts data structure to pipeline parameters
func IdentifyImportCandidates(_ context.Context, registries []registry.Registry, chartImageValuesMap ChartData, all bool, chartsOverview *myTable.Table, imagesOverview *myTable.Table, importEnabled bool) (RegistryChartStatus, RegistryImageStatus, error) {

	var sc counter.SafeCounter = counter.NewSafeCounter()

	c_header := table.Row{"#", "Helm Chart", "Chart Version"}
	c_footer := table.Row{"", "", ""}

	i_header := table.Row{"#", "Helm Chart", "Chart Version", "Image"}
	i_footer := table.Row{"", "", "", ""}

	// Create collection of registry names as keys for iterating registries
	keys := make([]string, 0)
	for _, r := range registries {
		keys = append(keys, r.URL)
	}

	// registry -> Charts -> bool
	m1 := make(RegistryChartStatus, 0)
	// registry -> Images -> bool
	m2 := make(RegistryImageStatus, 0)

	for c := range chartImageValuesMap {
		if c.Name == "images" {
			continue
		}

		// Charts
		n := fmt.Sprintf("charts/%s", c.Name)
		v := c.Version

		row := table.Row{sc.Value("index_import_charts"), c.Name, c.Version}

		for _, r := range registries {
			existsInRegistry := registry.Exists(context.TODO(), n, v, []registry.Registry{r})[r.URL]

			elem := m1[&r]
			if elem == nil {
				// init map
				elem = make(map[*Chart]bool, 0)
				m1[&r] = elem
			}
			b := all || !existsInRegistry
			elem[&c] = b

			if b {
				sc.Inc(r.URL + "charts")
			}
			row = append(row, terminal.StatusEmoji(existsInRegistry), terminal.StatusEmoji(b))
		}
		chartsOverview.AddRow(row)

		sc.Inc("index_import_charts")
	}

	var seenImages []registry.Image = make([]registry.Image, 0)
	for c, imageMap := range chartImageValuesMap {

		// Images
		for i := range imageMap {
			if i.In(seenImages) {
				ref, _ := i.String()
				log.Printf("Already parsed '%s', skipping...\n", ref)
				continue
			}
			// make sure we don't parse again
			seenImages = append(seenImages, *i)

			// decide if image should be imported
			name, err := i.ImageName()
			if err != nil {
				return nil, nil, err
			}

			// add row to overview table
			ref, _ := i.String()
			row := table.Row{sc.Value("index_import"), c.Name, c.Version, ref}

			for _, r := range registries {

				// check if image exists in registry
				registryImageStatusMap := registry.Exists(context.TODO(), name, i.Tag, []registry.Registry{r})
				// loop over registries
				imageExistsInRegistry := registryImageStatusMap[r.URL]

				row = append(row, terminal.StatusEmoji(imageExistsInRegistry))

				elem := m2[&r]
				if elem == nil {
					// init map
					elem = make(map[*registry.Image]bool, 0)
					m2[&r] = elem
				}
				b := all || !imageExistsInRegistry
				elem[i] = b

				if b {
					sc.Inc(r.URL)
				}
				row = append(row, terminal.StatusEmoji(b))

			}

			sc.Inc("index_import")
			imagesOverview.AddRow(row)
		}
	}

	// Table

	for _, r := range registries {

		// dynamic number of registries in table
		i_header = append(i_header, r.GetName())
		i_footer = append(i_footer, "")
		c_header = append(c_header, r.GetName())
		c_footer = append(c_footer, "")

		if importEnabled {
			// second static part of header
			i_header = append(i_header, "import")
			i_footer = append(i_footer, sc.Value(r.URL))
			c_header = append(c_header, "import")
			c_footer = append(c_footer, sc.Value(r.URL+"charts"))
		}
	}

	chartsOverview.AddHeader(c_header)
	chartsOverview.AddFooter(c_footer)
	imagesOverview.AddHeader(i_header)
	imagesOverview.AddFooter(i_footer)

	return m1, m2, nil
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
		charts := []*Chart{}

		for c, b := range m {
			if b {

				chartRef, err := c.ChartRef()
				// _, chartRef, _, err := c.Read(args.Update)
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

					// only import remote charts
					if d.Repository == "" || strings.HasPrefix(d.Repository, "file://") {
						// Embedded in parent chart
						slog.Debug("Skipping embedded chart", slog.String("chart", d.Name), slog.String("parent", c.Name))
						continue
					}

					chart := DependencyToChart(d, c)

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

			slog.Default().With(slog.String("chart", c.Name))

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
