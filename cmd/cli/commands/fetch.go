package commands

import (
	"context"
	"log"
	"log/slog"

	"github.com/ChristofferNissen/helmper/pkg/image"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/ChristofferNissen/helmper/internal/bootstrap"
	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"helm.sh/helm/v3/pkg/cli"
)

func init() {
	rootCmd.AddCommand(fetchCmd)
}

var fetchCmd = &cobra.Command{
	Use:     "fetch",
	Short:   "Fetch artifacts from Helm Chart(s)",
	GroupID: "remote",
	// Long:  `All software has versions. This is Helmper CLI's`,
	Run: func(cmd *cobra.Command, args []string) {
		done := make(chan error) // Channel to signal completion

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
						go func() error {

							var (
								k8sVersion   string                          = state.GetValue[string](v, "k8s_version")
								verbose      bool                            = state.GetValue[bool](v, "verbose")
								update       bool                            = state.GetValue[bool](v, "update")
								parserConfig bootstrap.ParserConfigSection   = state.GetValue[bootstrap.ParserConfigSection](v, "parserConfig")
								mirrorConfig []bootstrap.MirrorConfigSection = state.GetValue[[]bootstrap.MirrorConfigSection](v, "mirrorConfig")
								images       []image.Image                   = state.GetValue[[]image.Image](v, "images")
								charts       *helm.ChartCollection           = to.Ptr(state.GetValue[helm.ChartCollection](v, "input"))
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
							data, err := co.Run(ctx, opts...)
							if err != nil {
								return err
							}
							// Output overview table of charts and subcharts
							co.ChartTable.Render()
							// Output overview of helm path values per image
							co.ValueTable.Render()
							slog.Debug("Parsing of user specified chart(s) completed")

							path := "artifacts"
							err = helm.WriteYAMLToFile(path, data)
							if err != nil {
								done <- err
							}

							slog.Info("Successfully wrote artifacts file", slog.String("path", path))

							done <- nil
							return nil

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
			log.Fatal(err.Error())
		}

	},
}
