package commands

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/ChristofferNissen/helmper/pkg/copa"
	"github.com/ChristofferNissen/helmper/pkg/flow"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/trivy"

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
	rootCmd.AddCommand(importCmd)
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Push artifacts to registry",

	GroupID: "remote",
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
								k8sVersion   string                        = state.GetValue[string](v, "k8s_version")
								verbose      bool                          = state.GetValue[bool](v, "verbose")
								update       bool                          = state.GetValue[bool](v, "update")
								all          bool                          = state.GetValue[bool](v, "all")
								importConfig bootstrap.ImportConfigSection = state.GetValue[bootstrap.ImportConfigSection](v, "importConfig")
								opts         []helm.Option                 = []helm.Option{
									helm.K8SVersion(k8sVersion),
									helm.Verbose(verbose),
									helm.Update(update),
								}
							)

							if verbose {
								slog.SetLogLoggerLevel(slog.LevelDebug)
							}

							mCharts, mImgs, err := helm.ReadStatusOutputFromYAML("status.yaml")
							if err != nil {
								done <- err
							}

							// Step 4: Import charts to registries
							if importConfig.Import.Enabled {
								ctx := context.WithoutCancel(ctx)
								err := helm.ChartImportOption{
									Data:           mCharts,
									All:            all,
									ModifyRegistry: importConfig.Import.ReplaceRegistryReferences,

									Settings: settings,
								}.Run(ctx, opts...)
								if err != nil {
									done <- err
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
									done <- err
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
									done <- err
									return err
								}
							}

							slog.Info("finished import")
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
