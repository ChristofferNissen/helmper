package helm

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/bobg/go-generics/slices"
	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"

	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/report"
	"github.com/ChristofferNissen/helmper/pkg/util/bar"
	"github.com/ChristofferNissen/helmper/pkg/util/counter"
	"github.com/ChristofferNissen/helmper/pkg/util/terminal"
)

type ChartOption struct {
	ChartCollection     *ChartCollection
	IdentifyImages      bool
	UseCustomValues     bool
	FailOnMissingImages bool

	Mirrors []Mirror
	Images  []image.Image

	ChartTable *report.Table
	ValueTable *report.Table

	Settings *cli.EnvSettings
}

func determineTag(ctx context.Context, img *image.Image, plainHTTP bool) bool {
	ctx = context.WithoutCancel(ctx)
	ref := img.String()
	tag, err := img.TagOrDigest()
	if err != nil {
		return false
	}

	available, _ := registry.Exist(ctx, ref, tag, plainHTTP)
	if available {
		return true
	}

	available, _ = registry.Exist(ctx, ref, "v"+tag, plainHTTP)
	if available {
		img.Tag = "v" + img.Tag
		return true
	}

	return false
}

func determineSubChartPath(settings *cli.EnvSettings, d *chart.Dependency, subChart *Chart, path string, args *Options) (string, error) {
	p := path

	// Check if path is archive e.g. contains '.tgz'
	if strings.Contains(p, ".tgz") {
		// Unpack tar
		if err := chartutil.ExpandFile(settings.EnvVars()["HELM_CACHE_HOME"], p); err != nil {
			return "", err
		}

		p = filepath.Join(settings.EnvVars()["HELM_CACHE_HOME"], subChart.Parent.Name)
	}

	switch {
	case strings.HasPrefix(d.Repository, "file://"): //  Helm version >2.2.0
		fallthrough
	case d.Repository == "": // Embedded
		return fmt.Sprintf("%s/charts/%s", p, subChart.Name), nil
	}

	// Get Dependency Charts to local filesystem
	subChartPath, err := subChart.Locate(settings)
	if err != nil {
		return "", err
	}

	_ = updateRepository(
		settings,
		subChartPath,
		Verbose(args.Verbose),
		Update(args.Update),
	)

	_, _ = subChart.Pull(settings)
	// if err != nil {
	// 	return "", err
	// }

	return subChartPath, nil
}

func replaceValue(elem []string, new string, m map[string]interface{}) error {
	e, rest := elem[0], elem[1:]

	vm, ok := m[e].(map[string]interface{})
	if ok {
		return replaceValue(rest, new, vm)
	} else {
		switch m[e].(type) {
		case string:
			m[e] = new
			return nil
		default:
			return xerrors.New("could not replace value")
		}
	}
}

func replaceWithMirrors(cm *ChartData, mirrorConfig []Mirror) error {
	// modify images according to user specification
	for c, m := range *cm {
		for i, vs := range m {
			r := i.String()

			if c.Images != nil {
				for _, e := range c.Images.Exclude {
					if strings.HasPrefix(r, e.Ref) {
						delete(m, i)
						slog.Info("excluded image", slog.String("image", r))
						break
					}
				}
				for _, ec := range c.Images.ExcludeCopacetic {
					if strings.HasPrefix(r, ec.Ref) {
						slog.Info("excluded image from copacetic patching", slog.String("image", r))
						f := false
						i.Patch = &f
						break
					}
				}
				for _, modify := range c.Images.Modify {
					if modify.From != "" {
						if strings.HasPrefix(r, modify.From) {
							delete(m, i)

							img, err := image.RefToImage(
								strings.Replace(r, modify.From, modify.To, 1),
							)
							if err != nil {
								return err
							}

							img.Digest = i.Digest
							img.UseDigest = i.UseDigest
							img.Tag = i.Tag
							img.Patch = i.Patch

							m[&img] = vs

							newR := img.String()
							slog.Info("modified image reference", slog.String("old_image", r), slog.String("new_image", newR))
						}
					}
				}
			}

			// Replace mirrors
			ms, err := slices.Filter(mirrorConfig, func(m Mirror) (bool, error) {
				return m.Registry == i.Registry, nil
			})
			if err != nil {
				return err
			}

			if len(ms) > 0 {
				_ = i.ReplaceRegistry(ms[0].Mirror)
			}
		}
	}
	return nil
}

func (co *ChartOption) Run(ctx context.Context, setters ...Option) (ChartData, error) {
	// Default Options
	args := &Options{
		Verbose:    false,
		Update:     false,
		K8SVersion: "1.31.1",
	}

	for _, setter := range setters {
		setter(args)
	}

	if co.Settings == nil {
		co.Settings = cli.New()
	}

	// init tables
	if co.ChartTable == nil {
		co.ChartTable = report.NewTable("Charts")
	}
	if co.ValueTable == nil {
		co.ValueTable = report.NewTable("Helm Values Paths Per Image")
	}

	co.ChartTable.AddHeader(table.Row{"#", "Type", "Chart", "Version", "Latest Version", "Latest", "Values", "SubChart", "Version", "Condition", "Enabled"})
	co.ValueTable.AddHeader(table.Row{"#", "Helm Chart", "Chart Version", "Image", "Helm Value Path(s)"})

	sc := counter.NewSafeCounter()

	eg, egCtx := errgroup.WithContext(ctx)

	// Load chart from local filesystem and pass on information on channel
	chartGenerator := func(charts *ChartCollection) <-chan *chartInfo {
		channel := make(chan *chartInfo)

		eg.Go(func() error {
			defer close(channel)

			if len(charts.Charts) == 0 {
				// nothing to process
				return nil
			}

			bar := bar.New("Parsing charts...\r", len(charts.Charts))

			for _, c := range charts.Charts {
				slog.Default().With(slog.String("chart", c.Name), slog.String("repo", c.Repo.URL), slog.String("version", c.Version))

				// Check for latest version of chart
				latest, err := c.LatestVersion(co.Settings)
				if err != nil {
					slog.Error(err.Error())
					// continue as we do not want this failed lookup to stop the program
				}

				// Read info from filesystem
				path, chartRef, values, err := c.Read(co.Settings, args.Update)
				if err != nil {
					return err
				}
				valuesType := report.DeterminePathType(c.ValuesFilePath)

				bar.ChangeMax(bar.GetMax() + len(chartRef.Metadata.Dependencies))

				co.ChartTable.AddRow(table.Row{sc.Value("charts"), "Chart", c.Name, c.Version, latest, terminal.StatusEmoji(c.Version == latest), valuesType, "", "", "", ""})

				// reserve ids for table output
				sc.Inc("charts")
				count := len(chartRef.Metadata.Dependencies)
				reservedIDs := make([]int, count)
				for i := 0; i < count; i++ {
					reservedIDs[i] = sc.Value("charts")
					sc.Inc("charts")
				}

				_ = bar.Add(1)
				channel <- &chartInfo{chartRef, c}

				// Look at SubCharts if they are enabled (chart dependency condition satisfied in values.yaml)
				for id, d := range chartRef.Metadata.Dependencies {

					// subchart enabled in main chart?
					enabled := ConditionMet(d.Condition, values)
					if args.Verbose {
						log.Printf("Chart '%s' SubChart '%s' enabled by condition '%s': %t\n", chartRef.Name(), d.Name, d.Condition, enabled)
					}
					slog.Debug(
						"SubChart enabled by condition in parent chart",
						slog.String("subChartName", d.Name),
						slog.String("condition", d.Condition),
						slog.Bool("enabled", enabled))

					co.ChartTable.AddRow(table.Row{reservedIDs[id], "Subchart", "", "", "", "", "parent", d.Name, d.Version, d.Condition, terminal.StatusEmoji(enabled)})

					if d.Repository == "" || strings.HasPrefix(d.Repository, "file://") {
						_ = bar.Add(1)
						continue
					}

					// if condition is met to include subChart
					if !enabled {
						_ = bar.Add(1)
						continue
					}

					// Create chart for dependency
					subChart := DependencyToChart(d, c)
					v, err := subChart.ResolveVersion(co.Settings)
					if err != nil {
						return err
					}
					subChart.Version = v

					// Determine path to subChart in filesystem
					scPath, err := determineSubChartPath(co.Settings, d, subChart, path, args)
					if err != nil {
						return err
					}
					chartRef, err := loader.Load(scPath)
					if err != nil {
						return err
					}

					_ = bar.Add(1)
					channel <- &chartInfo{chartRef, subChart}
				}
			}

			return bar.Finish()
		})

		return channel
	}

	chartCollector := func(charts <-chan *chartInfo) ChartData {
		chartImageHelmValuesMap := make(ChartData)

		for c := range charts {
			chartImageHelmValuesMap[c.Chart] = nil
		}

		return chartImageHelmValuesMap
	}

	// Parse charts for images
	imageGenerator := func(charts <-chan *chartInfo) <-chan *imageInfo {
		channel := make(chan *imageInfo)

		eg.Go(func() error {
			defer close(channel)

			// find image references in charts and subcharts
			for chart := range charts {
				// Find images in Helm Chart (chart -> image -> helm properties)
				c, chart := chart.Chart, chart.chartRef

				// Get custom Helm values
				values, err := c.GetValues(co.Settings)
				if err != nil {
					return err
				}

				// Perform user customization
				if c.Images != nil {
					for _, mod := range c.Images.Modify {
						if mod.FromValuePath != "" {
							slog.Info("modifying chart value", slog.String("HelmValuesPath", mod.FromValuePath), slog.String("new", mod.To))
							to := mod.To
							versionToken := "{.version}"
							to = strings.Replace(to, versionToken, c.Version, 1)
							err := replaceValue(strings.Split(mod.FromValuePath, "."), to, chart.Values)
							if err != nil {
								return err
							}
						}
					}
				}

				// find images and validate according to values
				imageMap := findImageReferences(chart.Values, values, co.UseCustomValues)

				// check that images are available from registries
				if imageMap == nil {
					return nil
				}

				eg, egCtx := errgroup.WithContext(egCtx)
				for i, helmValuePaths := range imageMap {
					func(i *image.Image, helmValuePaths []string) {
						eg.Go(func() error {
							if i.IsEmpty() {
								return nil
							}

							// shuffle data (ensure all fields are populated in i)
							reg, repo, name, _ := i.Elements()
							i.Registry = reg
							i.Repository = fmt.Sprintf("%s/%s", repo, name)

							if i.Tag == "" {
								switch name {
								case "kubectl":
									i.Tag = args.K8SVersion
								default:
									// If tag is empty in values.yaml, use App Version by convention
									i.Tag = chart.Metadata.AppVersion
								}
							}

							plainHTTP := strings.Contains(i.Registry, "localhost") || strings.Contains(i.Registry, "0.0.0.0")
							available := determineTag(egCtx, i, plainHTTP)
							i.ResetParsedRef()

							// send availability response
							channel <- &imageInfo{available, c, i, &helmValuePaths}

							return nil
						})
					}(i, helmValuePaths)
				}

				err = eg.Wait()
				if err != nil {
					return err
				}

			}

			return nil
		})

		return channel
	}

	imageCollector := func(imgs <-chan *imageInfo) (ChartData, error) {
		chartImageHelmValuesMap := make(ChartData)
		id := 0

		for i := range imgs {
			if !i.available {
				str := i.image.String()
				slog.Info("Image not available. will be excluded from import...", slog.String("image", str))
				if co.FailOnMissingImages {
					return nil, xerrors.New("image not available")
				}
				continue
			}

			// Add Helm values to image map
			imageHelmValuesPathMap := make(map[*image.Image][]string)
			switch imageHelmValuesPathMap[i.image] {
			case nil:
				imageHelmValuesPathMap[i.image] = *i.collection
			default:
				imageHelmValuesPathMap[i.image] = append(imageHelmValuesPathMap[i.image], *i.collection...)
			}

			// Add table row
			ref := i.image.String()
			noSHA := strings.SplitN(ref, "@", 2)[0]
			co.ValueTable.AddRow(table.Row{id, i.chart.Name, i.chart.Version, noSHA, strings.Join(*i.collection, "\n")})

			// Add image map to chart map
			switch {
			case chartImageHelmValuesMap[i.chart] == nil:
				chartImageHelmValuesMap[i.chart] = imageHelmValuesPathMap
			case chartImageHelmValuesMap[i.chart][i.image] == nil:
				chartImageHelmValuesMap[i.chart][i.image] = imageHelmValuesPathMap[i.image]
			}

			id = id + 1
		}

		return chartImageHelmValuesMap, nil
	}

	workload := func(c *ChartCollection) (ChartData, error) {
		if co.IdentifyImages {
			return imageCollector(
				imageGenerator(
					chartGenerator(c),
				),
			)
		}
		return chartCollector(chartGenerator(c)), nil
	}

	cd, err := workload(co.ChartCollection)
	if err != nil {
		return nil, err
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// Replace mirrors for further processing
	err = replaceWithMirrors(&cd, co.Mirrors)
	if err != nil {
		return nil, err
	}

	if len(co.Images) > 0 {
		// Add in images from config
		placeHolder := &Chart{Name: "images", Version: "0.0.0"}
		m := map[*image.Image][]string{}
		for _, i := range co.Images {
			m[&i] = []string{}
		}
		cd[placeHolder] = m
	}

	// Make sure we parse Charts with no images as well
	for _, c := range co.ChartCollection.Charts {
		if cd[c] == nil {
			cd[c] = make(map[*image.Image][]string)
		}
	}

	return cd, nil
}
