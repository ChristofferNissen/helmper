package commands

import (
	"context"
	"log"
	"log/slog"

	"github.com/ChristofferNissen/helmper/pkg/registry"

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
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:     "status",
	GroupID: "remote",
	Short:   "Get status of artifacts in registry",
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
			fx.Invoke(func(lc fx.Lifecycle, v *viper.Viper, settings *cli.EnvSettings, rc helm.RegistryClient) {
				lc.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						go func() error {

							var (
								verbose      bool                          = state.GetValue[bool](v, "verbose")
								all          bool                          = state.GetValue[bool](v, "all")
								importConfig bootstrap.ImportConfigSection = state.GetValue[bootstrap.ImportConfigSection](v, "importConfig")
								registries   []*registry.Registry          = state.GetValue[[]*registry.Registry](v, "registries")
							)

							if verbose {
								slog.SetLogLoggerLevel(slog.LevelDebug)
							}

							data, err := helm.ReadYAMLFromFile("artifacts", rc)
							if err != nil {
								done <- err
							}

							log.Println(data)
							for c := range data {
								log.Println(c.Name)
							}

							// STEP 3: Validate and correct image references from charts
							slog.Debug("Checking presence of images from chart(s) in registries...")
							io := helm.IdentifyImportOption{
								Registries:          registries,
								ChartImageValuesMap: data,

								All:           all,
								ImportEnabled: importConfig.Import.Enabled,
							}
							mCharts, mImgs, err := io.Run(ctx)
							if err != nil {
								done <- err
								return err
							}
							io.ChartsOverview.Render()
							io.ImagesOverview.Render()
							slog.Debug("Checking presence of images from chart(s) in registries completed")

							err = helm.WriteStatusOutputToYAML("status.yaml", mCharts, mImgs)
							if err != nil {
								done <- err
							}

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
