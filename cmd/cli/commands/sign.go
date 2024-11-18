package commands

import (
	"context"
	"log"
	"log/slog"

	"github.com/ChristofferNissen/helmper/internal/bootstrap"
	mySign "github.com/ChristofferNissen/helmper/pkg/cosign"
	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"helm.sh/helm/v3/pkg/cli"
)

func init() {
	rootCmd.AddCommand(signCmd)
}

var signCmd = &cobra.Command{
	Use:     "sign",
	GroupID: "remote",
	Short:   "Sign artifacts in registry",
	Long: `The sign command allows you to sign Helm charts and container images stored in a registry.
It uses Cosign for signing and verification, ensuring the integrity and authenticity of your artifacts.`,
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
								importConfig bootstrap.ImportConfigSection = state.GetValue[bootstrap.ImportConfigSection](v, "importConfig")
							)

							if verbose {
								slog.SetLogLoggerLevel(slog.LevelDebug)
							}

							mCharts, mImgs, err := helm.ReadStatusOutputFromYAML("status.yaml")
							if err != nil {
								done <- err
								return err
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
									done <- err
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
									done <- err
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
									done <- err
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
									done <- err
									return err
								}
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
