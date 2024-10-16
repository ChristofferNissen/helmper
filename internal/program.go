package internal

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ChristofferNissen/helmper/internal/bootstrap"
	"github.com/ChristofferNissen/helmper/internal/output"
	"github.com/ChristofferNissen/helmper/pkg/copa"
	mySign "github.com/ChristofferNissen/helmper/pkg/cosign"
	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/trivy"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
	"github.com/bobg/go-generics/slices"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func modify(cm *helm.ChartData, mirrorConfig []bootstrap.MirrorConfigSection) error {
	// modify images according to user specification
	for c, m := range *cm {
		for i, vs := range m {
			r, err := i.String()
			if err != nil {
				return err
			}

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

							img, err := registry.RefToImage(
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

							newR, err := img.String()
							if err != nil {
								return err
							}
							slog.Info("modified image reference", slog.String("old_image", r), slog.String("new_image", newR))
						}
					}
				}
			}

			// Replace mirrors
			ms, err := slices.Filter(mirrorConfig, func(m bootstrap.MirrorConfigSection) (bool, error) {
				return m.Registry == i.Registry, nil
			})
			if err != nil {
				return err
			}

			if len(ms) > 0 {
				i.Registry = ms[0].Mirror
			}
		}
	}
	return nil
}

func Program(args []string) error {
	ctx := context.TODO()

	slogHandlerOpts := &slog.HandlerOptions{}
	if os.Getenv("HELMPER_LOG_LEVEL") == "DEBUG" {
		slogHandlerOpts.Level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, slogHandlerOpts))
	slog.SetDefault(logger)

	output.Header(version, commit, date)

	viper, err := bootstrap.LoadViperConfiguration(args)
	if err != nil {
		return err
	}
	var (
		k8sVersion   string                          = state.GetValue[string](viper, "k8s_version")
		verbose      bool                            = state.GetValue[bool](viper, "verbose")
		update       bool                            = state.GetValue[bool](viper, "update")
		all          bool                            = state.GetValue[bool](viper, "all")
		parserConfig bootstrap.ParserConfigSection   = state.GetValue[bootstrap.ParserConfigSection](viper, "parserConfig")
		importConfig bootstrap.ImportConfigSection   = state.GetValue[bootstrap.ImportConfigSection](viper, "importConfig")
		mirrorConfig []bootstrap.MirrorConfigSection = state.GetValue[[]bootstrap.MirrorConfigSection](viper, "mirrorConfig")
		registries   []registry.Registry             = state.GetValue[[]registry.Registry](viper, "registries")
		images       []registry.Image                = state.GetValue[[]registry.Image](viper, "images")
		charts       helm.ChartCollection            = state.GetValue[helm.ChartCollection](viper, "input")
		opts         []helm.Option                   = []helm.Option{
			helm.K8SVersion(k8sVersion),
			helm.Verbose(verbose),
			helm.Update(update),
		}
	)

	if verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	// Find input charts in configuration
	slog.Debug(
		"Found charts in config",
		slog.Int("count", len(charts.Charts)),
	)

	// STEP 1: Setup Helm
	charts, err = bootstrap.SetupHelm(
		&charts,
		opts...,
	)
	if err != nil {
		return err
	}
	// Output overview table of charts and subcharts
	go output.RenderChartTable(
		&charts,
		output.Update(update),
	)

	// STEP 2: Find images in Helm Charts and dependencies
	slog.Debug("Starting parsing user specified chart(s) for images..")
	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  !parserConfig.DisableImageDetection,
		UseCustomValues: parserConfig.UseCustomValues,
	}
	chartImageHelmValuesMap, err := co.Run(
		ctx,
		opts...,
	)
	if err != nil {
		return err
	}

	err = modify(&chartImageHelmValuesMap, mirrorConfig)
	if err != nil {
		return err
	}

	// Add in images from config
	placeHolder := helm.Chart{
		Name:    "images",
		Version: "0.0.0",
	}
	m := map[*registry.Image][]string{}
	for _, i := range images {
		m[&i] = []string{}
	}
	chartImageHelmValuesMap[placeHolder] = m

	// Output table of image to helm chart value path
	go func() {
		output.RenderHelmValuePathToImageTable(chartImageHelmValuesMap)
		slog.Debug("Parsing of user specified chart(s) completed")
	}()

	// STEP 3: Validate and correct image references from charts
	slog.Debug("Checking presence of images from chart(s) in registries...")
	mCharts, mImgs, err := helm.IdentifyImportCandidates(
		ctx,
		registries,
		chartImageHelmValuesMap,
		all,
	)
	if err != nil {
		return err
	}

	err = output.RenderChartOverviewTable(
		ctx,
		viper,
		len(charts.Charts),
		registries,
		charts,
	)
	if err != nil {
		return err
	}
	// Output table of image status in registries
	err = output.RenderImageOverviewTable(
		ctx,
		viper,
		len(mImgs),
		registries,
		chartImageHelmValuesMap,
	)
	if err != nil {
		return err
	}
	slog.Debug("Finished checking image availability in registries")

	// Import charts to registries
	if importConfig.Import.Enabled {
		err := helm.ChartImportOption{
			Data:           mCharts,
			All:            all,
			ModifyRegistry: importConfig.Import.ReplaceRegistryReferences,
		}.Run(ctx, opts...)
		if err != nil {
			return fmt.Errorf("internal: error importing chart to registry: %w", err)
		}
	}

	// Import images to registries
	switch {
	case importConfig.Import.Enabled && importConfig.Import.Copacetic.Enabled:
		slog.Debug("Import enabled and Copacetic enabled")
		err := copa.SpsOption{
			Data:         mImgs,
			Architecture: importConfig.Import.Architecture,

			ReportsFolder: importConfig.Import.Copacetic.Output.Reports.Folder,
			ReportsClean:  importConfig.Import.Copacetic.Output.Reports.Clean,
			TarsFolder:    importConfig.Import.Copacetic.Output.Tars.Folder,
			TarsClean:     importConfig.Import.Copacetic.Output.Tars.Clean,

			All: all,
			ScanOption: trivy.ScanOption{
				DockerHost:    importConfig.Import.Copacetic.Buildkitd.Addr,
				TrivyServer:   importConfig.Import.Copacetic.Trivy.Addr,
				Insecure:      importConfig.Import.Copacetic.Trivy.Insecure,
				IgnoreUnfixed: importConfig.Import.Copacetic.Trivy.IgnoreUnfixed,
				Architecture:  importConfig.Import.Architecture,
			},
			PatchOption: copa.PatchOption{
				Buildkit: struct {
					Addr       string
					CACertPath string
					CertPath   string
					KeyPath    string
				}{
					Addr:       importConfig.Import.Copacetic.Buildkitd.Addr,
					CACertPath: importConfig.Import.Copacetic.Buildkitd.CACertPath,
					CertPath:   importConfig.Import.Copacetic.Buildkitd.CertPath,
					KeyPath:    importConfig.Import.Copacetic.Buildkitd.KeyPath,
				},
				IgnoreErrors: importConfig.Import.Copacetic.IgnoreErrors,
				Architecture: importConfig.Import.Architecture,
			},
		}.Run(ctx)
		if err != nil {
			return err
		}

	case importConfig.Import.Enabled:
		slog.Debug("Import enabled")
		err := registry.ImportOption{
			Data:         mImgs,
			All:          all,
			Architecture: importConfig.Import.Architecture,
		}.Run(ctx)
		if err != nil {
			return err
		}
	}

	// Sign
	if importConfig.Import.Cosign.Enabled {

		// Charts

		vco := mySign.VerifyChartOption{
			Data:           mCharts,
			VerifyExisting: importConfig.Import.Cosign.VerifyExisting,

			KeyRef:            *importConfig.Import.Cosign.PubKeyRef,
			AllowInsecure:     importConfig.Import.Cosign.AllowInsecure,
			AllowHTTPRegistry: importConfig.Import.Cosign.AllowHTTPRegistry,
		}
		charts, err := vco.Run()
		if err != nil {
			return err
		}
		sco := mySign.SignChartOption{
			Data: charts,

			KeyRef:            importConfig.Import.Cosign.KeyRef,
			KeyRefPass:        *importConfig.Import.Cosign.KeyRefPass,
			AllowInsecure:     importConfig.Import.Cosign.AllowInsecure,
			AllowHTTPRegistry: importConfig.Import.Cosign.AllowHTTPRegistry,
		}
		if err := sco.Run(); err != nil {
			slog.Error("Error signing with Cosign")
			return err
		}

		// Images

		vo := mySign.VerifyOption{
			Data:           mImgs,
			VerifyExisting: importConfig.Import.Cosign.VerifyExisting,

			KeyRef:            *importConfig.Import.Cosign.PubKeyRef,
			AllowInsecure:     importConfig.Import.Cosign.AllowInsecure,
			AllowHTTPRegistry: importConfig.Import.Cosign.AllowHTTPRegistry,
		}
		imgs, err := vo.Run()
		if err != nil {
			return err
		}
		so := mySign.SignOption{
			Data: imgs,

			KeyRef:            importConfig.Import.Cosign.KeyRef,
			KeyRefPass:        *importConfig.Import.Cosign.KeyRefPass,
			AllowInsecure:     importConfig.Import.Cosign.AllowInsecure,
			AllowHTTPRegistry: importConfig.Import.Cosign.AllowHTTPRegistry,
		}
		if err := so.Run(); err != nil {
			return err
		}
	}

	return nil
}
