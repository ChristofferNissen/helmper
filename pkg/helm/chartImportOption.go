package helm

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sort"

	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/sync/errgroup"
	"helm.sh/helm/v3/pkg/cli"

	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/report"
	"github.com/ChristofferNissen/helmper/pkg/util/bar"
	"github.com/ChristofferNissen/helmper/pkg/util/counter"
	"github.com/ChristofferNissen/helmper/pkg/util/terminal"
)

type IdentifyImportOption struct {
	Registries          []*registry.Registry
	ChartImageValuesMap ChartData

	All           bool
	ImportEnabled bool

	ChartsOverview *report.Table
	ImagesOverview *report.Table
}

// Converts data structure to pipeline parameters
func (io *IdentifyImportOption) Run(_ context.Context) (RegistryChartStatus, RegistryImageStatus, error) {
	if io.ChartsOverview == nil {
		io.ChartsOverview = report.NewTable("Registry Overview For Charts")
	}
	if io.ImagesOverview == nil {
		io.ImagesOverview = report.NewTable("Registry Overview For Images")
	}

	var sc counter.SafeCounter = counter.NewSafeCounter()

	c_header := table.Row{"#", "Helm Chart", "Chart Version"}
	c_footer := table.Row{"", "", ""}

	i_header := table.Row{"#", "Helm Chart", "Chart Version", "Image"}
	i_footer := table.Row{"", "", "", ""}

	// registry -> Charts -> bool
	m1 := make(RegistryChartStatus, 0)
	// registry -> Images -> bool
	m2 := make(RegistryImageStatus, 0)

	for c := range io.ChartImageValuesMap {
		if c.Name == "images" {
			continue
		}

		// Charts
		n := fmt.Sprintf("charts/%s", c.Name)
		v := c.Version

		row := table.Row{sc.Value("index_import_charts"), fmt.Sprintf("charts/%s", c.Name), c.Version}

		for _, r := range io.Registries {
			elem := m1[r]
			if elem == nil {
				// init map
				elem = make(map[*Chart]bool, 0)
				m1[r] = elem
			}

			existsInRegistry := registry.Exists(context.TODO(), n, v, []*registry.Registry{r})[r.URL]
			b := io.All || !existsInRegistry
			elem[c] = b
			if b {
				sc.Inc(r.URL + "charts")
			}
			row = append(row, terminal.StatusEmoji(existsInRegistry), terminal.StatusEmoji(b))
		}
		io.ChartsOverview.AddRow(row)

		sc.Inc("index_import_charts")
	}

	var seenImages []image.Image = make([]image.Image, 0)
	for c, imageMap := range io.ChartImageValuesMap {
		// Images
		for i := range imageMap {
			if i.In(seenImages) {
				ref := i.String()
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
			ref := i.String()
			row := table.Row{sc.Value("index_import"), c.Name, c.Version, ref}

			for _, r := range io.Registries {
				if r.PrefixSource {
					old := name
					name, _ = image.UpdateNameWithPrefixSource(i)
					slog.Info("registry has PrefixSource enabled", slog.String("old", old), slog.String("new", name))
				}

				// check if image exists in registry
				registryImageStatusMap := registry.Exists(context.TODO(), name, i.Tag, []*registry.Registry{r})
				// loop over registries
				imageExistsInRegistry := registryImageStatusMap[r.URL]

				row = append(row, terminal.StatusEmoji(imageExistsInRegistry))

				elem := m2[r]
				if elem == nil {
					// init map
					elem = make(map[*image.Image]bool, 0)
					m2[r] = elem
				}
				b := io.All || !imageExistsInRegistry
				elem[i] = b

				if b {
					sc.Inc(r.URL)
				}
				row = append(row, terminal.StatusEmoji(b))
			}

			sc.Inc("index_import")
			io.ImagesOverview.AddRow(row)
		}
	}

	// Table
	for _, r := range io.Registries {
		// dynamic number of registries in table
		rn := r.GetName()
		c_header = append(c_header, rn)
		c_footer = append(c_footer, "")
		i_header = append(i_header, rn)
		i_footer = append(i_footer, "")

		if io.ImportEnabled {
			// second static part of header
			c_header = append(c_header, "import")
			c_footer = append(c_footer, sc.Value(r.URL+"charts"))
			i_header = append(i_header, "import")
			i_footer = append(i_footer, sc.Value(r.URL))
		}
	}

	io.ChartsOverview.AddHeader(c_header)
	io.ChartsOverview.AddFooter(c_footer)
	io.ImagesOverview.AddHeader(i_header)
	io.ImagesOverview.AddFooter(i_footer)

	return m1, m2, nil
}

type ChartImportOption struct {
	Data           RegistryChartStatus
	All            bool
	ModifyRegistry bool

	Settings *cli.EnvSettings
}

func (opt ChartImportOption) Run(ctx context.Context, setters ...Option) error {
	// Default Options
	args := &Options{
		Verbose:    false,
		Update:     false,
		K8SVersion: "1.31.1",
	}

	for _, setter := range setters {
		setter(args)
	}

	if opt.Settings == nil {
		opt.Settings = cli.New()
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

	if size <= 0 {
		return nil
	}

	bar := bar.New("Pushing charts...\r", size)

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		eg, egCtx := errgroup.WithContext(egCtx)

		for r, m := range opt.Data {
			charts := []*Chart{}

			eg.Go(func() error {
				for c, b := range m {
					if b {
						chartRef, err := c.ChartRef(opt.Settings)
						if err != nil {
							return err
						}

						c.DepsCount = len(chartRef.Metadata.Dependencies)
						charts = append(charts, c)
					}
				}

				// Sort charts according to least dependencies
				sort.Slice(charts, func(i, j int) bool {
					return charts[i].DepsCount < charts[j].DepsCount
				})

				for _, c := range charts {

					// scope
					c := c

					eg.Go(func() error {
						slog.With(slog.String("chart", c.Name))

						if c.Name == "images" {
							return nil
						}

						if !opt.All {
							_, err := r.Exist(egCtx, "charts/"+c.Name, c.Version)
							if err == nil {
								slog.Info("Chart already present in registry. Skipping import", slog.String("chart", "charts/"+c.Name), slog.String("registry", r.URL), slog.String("version", c.Version))
								return nil
							}
							slog.Debug(err.Error())
						}

						if opt.ModifyRegistry {
							res, err := c.PushAndModify(opt.Settings, r.URL, r.Insecure, r.PlainHTTP, r.PrefixSource)
							if err != nil {
								return fmt.Errorf("helm: error pushing and modifying chart %s to registry %s :: %w", c.Name, r.URL, err)
							}
							slog.Debug(res)
							defer os.RemoveAll(res)
							_ = bar.Add(1)
							return nil
						}

						client, err := NewRegistryClient(r.PlainHTTP, false)
						if err != nil {
							return fmt.Errorf("helm: error creating registry client :: %w", err)
						}
						c.RegistryClient = client
						res, err := c.Push(opt.Settings, r.URL, r.Insecure, r.PlainHTTP)
						if err != nil {
							return fmt.Errorf("helm: error pushing chart %s to registry %s :: %w", c.Name, r.URL, err)
						}
						slog.Debug(res)

						_ = bar.Add(1)

						return nil
					})
				}

				return nil
			})
		}

		err := eg.Wait()
		if err != nil {
			return err
		}

		return nil
	})
	err := eg.Wait()
	if err != nil {
		return err
	}

	return bar.Finish()
}
