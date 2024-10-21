package internal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ChristofferNissen/helmper/internal/bootstrap"
	"github.com/ChristofferNissen/helmper/internal/output"
	"github.com/ChristofferNissen/helmper/pkg/copa"
	mySign "github.com/ChristofferNissen/helmper/pkg/cosign"
	"github.com/ChristofferNissen/helmper/pkg/flow"
	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/myTable"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/trivy"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func Program(args []string) error {
	done := make(chan error) // Channel to signal completion

	app := fx.New(
		helm.RegistryModule,
		bootstrap.ViperModule,
		LoggerModule,
		fx.Invoke(func(logger *slog.Logger) {
			// Logger is set up and can be used here
			logger.Info("Logger is configured")
		}),
		fx.Invoke(func(lc fx.Lifecycle, v *viper.Viper) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						done <- program(ctx, args, v) // Send the result to the channel
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

func program(ctx context.Context, _ []string, viper *viper.Viper) error {

	output.Header(version, commit, date)

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
	slog.Debug("Found charts in config", slog.Int("count", len(charts.Charts)))

	// STEP 1: Setup Helm
	charts, err := SetupHelm(&charts, opts...)
	if err != nil {
		return err
	}

	// STEP 2: Find images in Helm Charts and dependencies
	chartTable := myTable.NewTable("Charts")
	chartTable.AddHeader(table.Row{"#", "Type", "Chart", "Version", "Latest Version", "Latest", "Values", "SubChart", "Version", "Condition", "Enabled"})
	valueTable := myTable.NewTable("Helm Values Paths Per Image")
	valueTable.AddHeader(table.Row{"#", "Helm Chart", "Chart Version", "Image", "Helm Value Path(s)"})
	slog.Debug("Starting parsing user specified chart(s) for images..")
	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  !parserConfig.DisableImageDetection,
		UseCustomValues: parserConfig.UseCustomValues,

		Mirrors: bootstrap.ConvertToHelmMirrors(mirrorConfig),
		Images:  images,

		ChartTable: chartTable,
		ValueTable: valueTable,
	}
	chartImageHelmValuesMap, err := co.Run(ctx, opts...)
	if err != nil {
		return err
	}
	// Output overview table of charts and subcharts
	chartTable.Render()
	valueTable.Render()
	slog.Debug("Parsing of user specified chart(s) completed")

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
	// Output table of chart status in registries
	err = output.RenderChartOverviewTable(
		context.WithoutCancel(ctx),
		viper,
		len(charts.Charts),
		registries,
		charts,
	)
	if err != nil {
		return err
	}
	slog.Debug("Finished checking charts availability in registries")
	// Output table of image status in registries
	err = output.RenderImageOverviewTable(
		context.WithoutCancel(ctx),
		viper,
		len(mImgs),
		registries,
		chartImageHelmValuesMap,
	)
	if err != nil {
		return err
	}
	slog.Debug("Finished checking image availability in registries")

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
