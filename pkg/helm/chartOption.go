package helm

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

type ChartData map[Chart]map[*registry.Image][]string

// Converts data structure to pipeline parameters
func IdentifyImportCandidates(ctx context.Context, registries []registry.Registry, chartImageValuesMap ChartData, all bool) ([]registry.Image, error) {

	// Combine results
	imgs := make([]registry.Image, 0)
	var seenImages []registry.Image = make([]registry.Image, 0)

	for _, imageMap := range chartImageValuesMap {
		for i := range imageMap {
			if i.In(seenImages) {
				ref, _ := i.String()
				log.Printf("Already parsed '%s', skipping...\n", ref)
				continue
			}
			// make sure we don't parse again
			seenImages = append(seenImages, *i)

			// decide if image should be imported
			if all || func(rs []registry.Registry) bool {
				importImage := false
				// check if image exists in registry
				registryImageStatusMap, _ := registry.Exists(ctx, i, rs)
				// loop over registries
				for _, r := range rs {
					imageExistsInRegistry := registryImageStatusMap[r.GetName()]
					importImage = importImage || !imageExistsInRegistry
				}
				return importImage
			}(registries) {
				imgs = append(imgs, *i)
			}

		}
	}

	return imgs, nil
}

// channels to share data between goroutines
type chartInfo struct {
	chartRef *chart.Chart
	*Chart
}

type imageInfo struct {
	available  bool
	chart      *Chart
	image      *registry.Image
	collection *[]string
}

type ChartOption struct {
	ChartCollection *ChartCollection
}

func determineTag(ctx context.Context, k8sv string, img *registry.Image, plainHTTP bool) bool {

	reg, repo, name := img.Elements()
	ref := fmt.Sprintf("%s/%s/%s", reg, repo, name)

	tag, _ := img.TagOrDigest()

	available, _ := registry.Exist(ctx, ref, tag, plainHTTP)
	if available {
		return true
	}

	available, _ = registry.Exist(ctx, ref, "v"+img.Tag, plainHTTP)
	if available {
		img.Tag = "v" + img.Tag
		return true
	}

	available, _ = registry.Exist(ctx, ref, k8sv, plainHTTP)
	if available {
		img.Tag = k8sv
		return true
	}

	return false
}

func determineSubChartPath(d *chart.Dependency, subChart *Chart, c *Chart, path string, args *Options) (string, error) {
	if d.Repository == "" {
		p := path

		// Check if path is archive e.g. contains '.tgz'
		if strings.Contains(p, ".tgz") {
			// Unpack tar
			if err := chartutil.ExpandFile(cli.New().EnvVars()["HELM_CACHE_HOME"], p); err != nil {
				return "", err
			}
			p = filepath.Join(cli.New().EnvVars()["HELM_CACHE_HOME"], c.Name)
		}

		return fmt.Sprintf("%s/charts/%s", p, subChart.Name), nil
	}

	// Get Dependency Charts to local filesystem
	subChartPath, err := subChart.Locate()
	if err != nil {
		return "", err
	}

	_ = updateRepository(
		subChartPath,
		Verbose(args.Verbose),
		Update(args.Update),
	)

	_, _ = subChart.Pull()

	return subChartPath, nil
}

func (co ChartOption) Run(ctx context.Context, setters ...Option) (ChartData, error) {

	// Default Options
	args := &Options{
		Verbose:    false,
		Update:     false,
		K8SVersion: "1.27.7",
	}

	for _, setter := range setters {
		setter(args)
	}

	eg, egCtx := errgroup.WithContext(ctx)

	// Load chart from local filesystem and pass on information on channel
	chartGenerator := func(charts *ChartCollection) <-chan *chartInfo {
		channel := make(chan *chartInfo)

		eg.Go(func() error {
			defer close(channel)

			bar := progressbar.NewOptions(len(charts.Charts),
				progressbar.OptionSetWriter(ansi.NewAnsiStdout()), // "github.com/k0kubun/go-ansi"
				progressbar.OptionEnableColorCodes(true),
				progressbar.OptionShowCount(),
				progressbar.OptionOnCompletion(func() {
					fmt.Fprint(os.Stderr, "\n")
				}),
				progressbar.OptionSetWidth(15),
				progressbar.OptionSetElapsedTime(true),
				progressbar.OptionSetDescription("Parsing charts...\r"),
				progressbar.OptionShowDescriptionAtLineEnd(),
				progressbar.OptionSetTheme(progressbar.Theme{
					Saucer:        "[green]=[reset]",
					SaucerHead:    "[green]>[reset]",
					SaucerPadding: " ",
					BarStart:      "[",
					BarEnd:        "]",
				}))

			for _, c := range charts.Charts {

				path, chartRef, values, err := c.Read(args.Update)
				if err != nil {
					return err
				}
				bar.ChangeMax(bar.GetMax() + len(chartRef.Metadata.Dependencies))

				_ = bar.Add(1)
				channel <- &chartInfo{chartRef, &c}

				// Look at SubCharts if they are enabled (chart dependency condition satisfied in values.yaml)
				for _, d := range chartRef.Metadata.Dependencies {

					// subchart enabled in main chart?
					enabled := ConditionMet(d.Condition, values)
					if args.Verbose {
						log.Printf("Chart '%s' SubChart '%s' enabled by condition '%s': %t\n", chartRef.Name(), d.Name, d.Condition, enabled)
					}

					// if condition is met to include subChart
					if !enabled {
						_ = bar.Add(1)
						continue
					}

					// Create chart for dependency
					subChart := Chart{
						Name: d.Name,
						Repo: repo.Entry{
							Name: c.Repo.Name,
							URL:  d.Repository,
						},
						Version: d.Version,
						Parent:  &c,
					}

					// Determine path to subChart in filesystem
					scPath, err := determineSubChartPath(d, &subChart, &c, path, args)
					if err != nil {
						return err
					}
					chartRef, err := loader.Load(scPath)
					if err != nil {
						return err
					}

					_ = bar.Add(1)
					channel <- &chartInfo{chartRef, &subChart}

				}
			}

			return bar.Finish()

		})

		return channel
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
				values, err := c.Values()
				if err != nil {
					return err
				}

				// find images and validate according to values
				imageMap := findImageReferences(chart.Values, values)

				// check that images are available from registries
				if imageMap == nil {
					return nil
				}

				eg, egCtx := errgroup.WithContext(egCtx)
				for i, helmValuePaths := range imageMap {
					if i.Tag == "" {
						// If tag is empty in values.yaml, use App Version by convention
						i.Tag = chart.Metadata.AppVersion
					}

					func(i *registry.Image, helmValuePaths []string) {
						eg.Go(func() error {

							// shuffle data (ensure all fields are populated in i)
							reg, repo, name := i.Elements()
							i.Registry = reg
							i.Repository = fmt.Sprintf("%s/%s", repo, name)

							plainHTTP := strings.Contains(i.Registry, "localhost") || strings.Contains(i.Registry, "0.0.0.0")

							available := determineTag(egCtx, args.K8SVersion, i, plainHTTP)

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

	collector := func(imgs <-chan *imageInfo) ChartData {
		chartImageHelmValuesMap := make(ChartData)

		for i := range imgs {
			if !i.available {
				slog.Info("Image not available. will be excluded from import...", slog.Any("image", i))
				continue
			}

			// Add Helm values to image map
			imageHelmValuesPathMap := make(map[*registry.Image][]string)
			switch imageHelmValuesPathMap[i.image] {
			case nil:
				imageHelmValuesPathMap[i.image] = *i.collection
			default:
				imageHelmValuesPathMap[i.image] = append(imageHelmValuesPathMap[i.image], *i.collection...)
			}

			// Add image map to chart map
			switch {
			case chartImageHelmValuesMap[*i.chart] == nil:
				chartImageHelmValuesMap[*i.chart] = imageHelmValuesPathMap
			case chartImageHelmValuesMap[*i.chart][i.image] == nil:
				chartImageHelmValuesMap[*i.chart][i.image] = imageHelmValuesPathMap[i.image]
			}
		}

		return chartImageHelmValuesMap
	}

	cd := collector(
		imageGenerator(
			chartGenerator(co.ChartCollection),
		),
	)

	if err := eg.Wait(); err != nil {
		return ChartData{}, err
	}

	return cd, nil
}
