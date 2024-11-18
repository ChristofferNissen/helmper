package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChristofferNissen/helmper/internal/bootstrap"
	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"oras.land/oras-go/v2/content/oci"
)

func init() {
	rootCmd.AddCommand(importTarsCmd)
}

var importTarsCmd = &cobra.Command{
	Use:   "importTars",
	Short: "Push tars to registry",
	// Long:  `All software has versions. This is Helmper CLI's`,
	GroupID: "local",
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
						go func() {

							// Extract to flags
							dest := "/tmp/helmper-export/"

							var (
								verbose    bool                 = state.GetValue[bool](v, "verbose")
								registries []*registry.Registry = state.GetValue[[]*registry.Registry](v, "registries")
							)

							if verbose {
								slog.SetLogLoggerLevel(slog.LevelDebug)
							}

							indexFilePath := filepath.Join(dest, "index.json")
							indexFile, err := os.Open(indexFilePath)
							if err != nil {
								done <- err
								return
							}
							defer indexFile.Close()

							var entries []importEntry
							decoder := json.NewDecoder(indexFile)
							if err := decoder.Decode(&entries); err != nil {
								done <- err
								return
							}

							for _, entry := range entries {
								slog.Info("Processing entry", "name", entry.Name, "path", entry.FilePath)

								filePath := filepath.Join(dest, entry.FilePath)
								slog.Info("Processing file", "file", filePath)

								// Fix logic by introducing field in index about type of artifact
								switch entry.Type {
								case "HELM CHART":

									// Validate Helm Chart

									file, err := os.Open(filePath)
									if err != nil {
										done <- err
										return
									}
									defer file.Close()

									bf, err := loader.LoadArchiveFiles(file)
									if err != nil {
										done <- err
										return
									}

									chart, err := loader.LoadFiles(bf)
									if err != nil {
										done <- err
										return
									}

									err = chart.Validate()
									if err != nil {
										done <- err
										return
									}

									bs, err := os.ReadFile(filePath)
									if err != nil {
										done <- fmt.Errorf("error reading chart: %w", err)
										return
									}

									for _, r := range registries {
										slog.Info("Pushing chart to registry", "registry", r.Name)

										if strings.HasPrefix(r.URL, "oci://") {
											slog.Debug("Creating OCI Registry client", slog.String("chart", entry.Name))
											local := strings.Contains(r.URL, "localhost") || strings.Contains(r.URL, "0.0.0.0") || strings.Contains(r.URL, "127.0.0.1")
											rc, _ = helm.NewRegistryClient(local, false)
											rc = helm.NewOCIRegistryClient(rc, local)
										}

										url := r.URL
										d := url + "/" + entry.Name + ":" + entry.Tag
										d, _ = strings.CutPrefix(d, "oci://")
										_, err = rc.Push(bs, d)
										if err != nil {
											done <- err
											return
										}
										slog.Info("Successfully pushed chart to registry", "registry", r.Name, "chart", entry.Name, "tag", entry.Tag)
									}

								case "OCI ARTIFACT":
									store, err := oci.NewFromTar(ctx, filePath)
									if err != nil {
										done <- err
										return
									}
									for _, r := range registries {
										slog.Debug("Pushing to registry", "registry", r.Name)
										_, err := r.PushTar(ctx, store, entry.Name, entry.Tag, nil)
										if err != nil {
											slog.Error("Failed to push tar", slog.Any("error", err))
											done <- err
											return
										}
										slog.Info("Successfully pushed OCI artifact to registry", "registry", r.Name, "artifact", entry.Name, "tag", entry.Tag)
									}
								}

							}

							// // Step 4: Import charts to registries
							// if importConfig.Import.Enabled {
							// 	ctx := context.WithoutCancel(ctx)
							// 	err := helm.ChartImportOption{
							// 		Data:           mCharts,
							// 		All:            all,
							// 		ModifyRegistry: importConfig.Import.ReplaceRegistryReferences,

							// 		Settings: settings,
							// 	}.Run(ctx, opts...)
							// 	if err != nil {
							// 		done <- err
							// 		return fmt.Errorf("internal: error importing chart to registry: %w", err)
							// 	}
							// }

							// // Step 5: Import images to registries
							// switch {
							// case importConfig.Import.Enabled && importConfig.Import.Copacetic.Enabled:
							// 	slog.Debug("Import enabled and Copacetic enabled")
							// 	err := flow.SpsOption{
							// 		Data:         mImgs,
							// 		Architecture: importConfig.Import.Architecture,

							// 		ReportsFolder: importConfig.Import.Copacetic.Output.Reports.Folder,
							// 		ReportsClean:  importConfig.Import.Copacetic.Output.Reports.Clean,
							// 		TarsFolder:    importConfig.Import.Copacetic.Output.Tars.Folder,
							// 		TarsClean:     importConfig.Import.Copacetic.Output.Tars.Clean,

							// 		All: all,
							// 		ScanOption: trivy.ScanOption{
							// 			DockerHost:    importConfig.Import.Copacetic.Buildkitd.Addr,
							// 			TrivyServer:   importConfig.Import.Copacetic.Trivy.Addr,
							// 			Insecure:      importConfig.Import.Copacetic.Trivy.Insecure,
							// 			IgnoreUnfixed: importConfig.Import.Copacetic.Trivy.IgnoreUnfixed,
							// 			Architecture:  importConfig.Import.Architecture,
							// 		},
							// 		PatchOption: copa.PatchOption{
							// 			Buildkit: struct {
							// 				Addr       string
							// 				CACertPath string
							// 				CertPath   string
							// 				KeyPath    string
							// 			}{
							// 				Addr:       importConfig.Import.Copacetic.Buildkitd.Addr,
							// 				CACertPath: importConfig.Import.Copacetic.Buildkitd.CACertPath,
							// 				CertPath:   importConfig.Import.Copacetic.Buildkitd.CertPath,
							// 				KeyPath:    importConfig.Import.Copacetic.Buildkitd.KeyPath,
							// 			},
							// 			IgnoreErrors: importConfig.Import.Copacetic.IgnoreErrors,
							// 			Architecture: importConfig.Import.Architecture,
							// 		},
							// 	}.Run(ctx)
							// 	if err != nil {
							// 		done <- err
							// 		return err
							// 	}
							// case importConfig.Import.Enabled:
							// 	slog.Debug("Import enabled")
							// 	ctx := context.WithoutCancel(ctx)
							// 	err := registry.ImportOption{
							// 		Data:         mImgs,
							// 		All:          all,
							// 		Architecture: importConfig.Import.Architecture,
							// 	}.Run(ctx)
							// 	if err != nil {
							// 		done <- err
							// 		return err
							// 	}
							// }

							slog.Info("finished import")
							done <- nil

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
