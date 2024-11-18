package commands

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ChristofferNissen/helmper/internal/bootstrap"
	"github.com/ChristofferNissen/helmper/pkg/copa"
	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/trivy"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"helm.sh/helm/v3/pkg/cli"
)

func init() {
	rootCmd.AddCommand(patchCmd)
}

var patchCmd = &cobra.Command{
	Use: "patch",
	// Short: "Push artifacts to registry",

	GroupID: "local",
	Run: func(cmd *cobra.Command, args []string) {
		done := make(chan error) // Channel to signal completion

		app := fx.New(
			fx.StartTimeout(1*time.Hour),
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

							// Extract to flags
							dest := "/tmp/helmper-export/"

							var (
								// k8sVersion   string                        = state.GetValue[string](v, "k8s_version")
								verbose bool = state.GetValue[bool](v, "verbose")
								// update       bool                          = state.GetValue[bool](v, "update")
								all          bool                          = state.GetValue[bool](v, "all")
								importConfig bootstrap.ImportConfigSection = state.GetValue[bootstrap.ImportConfigSection](v, "importConfig")
								// opts         []helm.Option                 = []helm.Option{
								// 	helm.K8SVersion(k8sVersion),
								// 	helm.Verbose(verbose),
								// 	helm.Update(update),
								// }
							)

							if verbose {
								slog.SetLogLoggerLevel(slog.LevelDebug)
							}

							indexFilePath := filepath.Join(dest, "index.json")
							indexFile, err := os.Open(indexFilePath)
							if err != nil {
								done <- err
							}
							defer indexFile.Close()

							var entries []importEntry
							decoder := json.NewDecoder(indexFile)
							if err := decoder.Decode(&entries); err != nil {
								done <- err
							}

							for _, entry := range entries {
								slog.Info("Processing entry", "name", entry.Name, "path", entry.FilePath)
							}

							err = SpsLocalOption{
								Data: entries,

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
									Local:         true,
								},
								PatchOption: copa.LocalPatchOption{
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

							slog.Info("finished patch")
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
