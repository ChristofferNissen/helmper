package internal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/common-nighthawk/go-figure"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"helm.sh/helm/v3/pkg/cli"

	"github.com/ChristofferNissen/helmper/internal/bootstrap"
	"github.com/ChristofferNissen/helmper/pkg/copa"
	mySign "github.com/ChristofferNissen/helmper/pkg/cosign"
	"github.com/ChristofferNissen/helmper/pkg/flow"
	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/trivy"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
	"github.com/ChristofferNissen/helmper/pkg/util/terminal"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func Header(version, commit, date string) {
	myFigure := figure.NewFigure("helmper", "rectangles", true)
	myFigure.Print()
	terminal.PrintYellow(fmt.Sprintf("version %s (commit %s, built at %s)\n", version, commit, date))
}

func Program(args []string) error {
	done := make(chan error) // Channel to signal completion

	Header(version, commit, date)

	app := fx.New(
		helm.RegistryModule,
		bootstrap.ViperModule,
		bootstrap.LoggerModule,
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			logger.Info("Logger is configured")
			return &fxevent.SlogLogger{
				Logger: logger,
			}
		}),
		bootstrap.HelmSettingsModule,
		fx.Invoke(func(lc fx.Lifecycle, v *viper.Viper, settings *cli.EnvSettings) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						done <- program(ctx, args, v, settings) // Send the result to the channel
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					return nil
				},
			})
		}),
	)

	go func() {
		app.Run() // Run the Fx app in a separate goroutine
		close(done)
	}()

	// Wait for the program to signal completion
	if err := <-done; err != nil {
		return err
	}

	return nil
}

func program(ctx context.Context, _ []string, viper *viper.Viper, settings *cli.EnvSettings) error {
	slog.Info("Helmper", slog.String("version", version), slog.String("commit", commit), slog.String("date", date))

	var (
		k8sVersion   string                          = state.GetValue[string](viper, "k8s_version")
		verbose      bool                            = state.GetValue[bool](viper, "verbose")
		update       bool                            = state.GetValue[bool](viper, "update")
		all          bool                            = state.GetValue[bool](viper, "all")
		parserConfig bootstrap.ParserConfigSection   = state.GetValue[bootstrap.ParserConfigSection](viper, "parserConfig")
		importConfig bootstrap.ImportConfigSection   = state.GetValue[bootstrap.ImportConfigSection](viper, "importConfig")
		mirrorConfig []bootstrap.MirrorConfigSection = state.GetValue[[]bootstrap.MirrorConfigSection](viper, "mirrorConfig")
		registries   []*registry.Registry            = state.GetValue[[]*registry.Registry](viper, "registries")
		images       []image.Image                   = state.GetValue[[]image.Image](viper, "images")
		charts       *helm.ChartCollection           = to.Ptr(state.GetValue[helm.ChartCollection](viper, "input"))
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
	slog.Debug("Found charts in config", slog.Int("count", len(charts.Charts)))

	// STEP 1: Setup Helm
	charts, err := bootstrap.SetupHelm(settings, charts, opts...)
	if err != nil {
		return err
	}

	// STEP 2: Find images in Helm Charts and dependencies
	slog.Debug("Starting parsing user specified chart(s) for images..")
	co := helm.ChartOption{
		ChartCollection: charts,
		IdentifyImages:  !parserConfig.DisableImageDetection,
		UseCustomValues: parserConfig.UseCustomValues,

		Mirrors: bootstrap.ConvertToHelmMirrors(mirrorConfig),
		Images:  images,
	}
	chartImageHelmValuesMap, err := co.Run(ctx, opts...)
	if err != nil {
		return err
	}
	// Output overview table of charts and subcharts
	co.ChartTable.Render()
	// Output overview of helm path values per image
	co.ValueTable.Render()
	slog.Debug("Parsing of user specified chart(s) completed")

	// STEP 3: Validate and correct image references from charts
	slog.Debug("Checking presence of images from chart(s) in registries...")
	io := helm.IdentifyImportOption{
		Registries:          registries,
		ChartImageValuesMap: chartImageHelmValuesMap,

		All:           all,
		ImportEnabled: importConfig.Import.Enabled,
	}
	mCharts, mImgs, err := io.Run(ctx)
	if err != nil {
		return err
	}
	io.ChartsOverview.Render()
	io.ImagesOverview.Render()
	slog.Debug("Checking presence of images from chart(s) in registries completed")

	// Step 4: Import charts to registries
	if importConfig.Import.Enabled {
		ctx := context.WithoutCancel(ctx)
		err := helm.ChartImportOption{
			Data:           mCharts,
			All:            all,
			ModifyRegistry: importConfig.Import.ReplaceRegistryReferences,
		}.Run(ctx, opts...)
		if err != nil {
			return fmt.Errorf("internal: error importing chart to registry: %w", err)
		}
	}

	// Step 5: Import images to registries
	switch {
	case importConfig.Import.Enabled && importConfig.Import.Copacetic.Enabled:
		slog.Debug("Import enabled and Copacetic enabled")
		err := flow.SpsOption{
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
		ctx := context.WithoutCancel(ctx)
		err := registry.ImportOption{
			Data:         mImgs,
			All:          all,
			Architecture: importConfig.Import.Architecture,
		}.Run(ctx)
		if err != nil {
			return err
		}
	}

	// Step 6: Sign
	if importConfig.Import.Cosign.Enabled {
		// Charts
		vco := mySign.VerifyChartOption{
			Data:           mCharts,
			VerifyExisting: importConfig.Import.Cosign.VerifyExisting,

			KeyRef:            *importConfig.Import.Cosign.PubKeyRef,
			AllowInsecure:     importConfig.Import.Cosign.AllowInsecure,
			AllowHTTPRegistry: importConfig.Import.Cosign.AllowHTTPRegistry,
		}
		charts, err := vco.Run(context.WithoutCancel(ctx))
		if err != nil {
			return err
		}
		vco.Report.Render()
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
		imgs, err := vo.Run(context.WithoutCancel(ctx))
		if err != nil {
			return err
		}
		vo.Report.Render()
		so := mySign.SignOption{
			Data: imgs,

			KeyRef:            importConfig.Import.Cosign.KeyRef,
			KeyRefPass:        *importConfig.Import.Cosign.KeyRefPass,
			AllowInsecure:     importConfig.Import.Cosign.AllowInsecure,
			AllowHTTPRegistry: importConfig.Import.Cosign.AllowHTTPRegistry,
		}
		if err := so.Run(ctx); err != nil {
			return err
		}
	}

	return nil
}
