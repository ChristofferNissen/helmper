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
	"time"

	"github.com/ChristofferNissen/helmper/internal/bootstrap"
	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/util/file"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
	"github.com/distribution/reference"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
)

func init() {
	rootCmd.AddCommand(exportCmd)
}

type importEntry struct {
	Registry string `json:"registry"`
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	FilePath string `json:"filePath"`
	Type     string `json:"type"`
}

var exportCmd = &cobra.Command{
	Use:     "export",
	GroupID: "local",
	Short:   "Export artifacts to a directory",
	Long:    `The export command allows you to export Helm charts and OCI images from a registry to a local tar archive. This can be useful for backup or offline usage.`,
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
						go func() {
							index := make([]*importEntry, 0)

							// Extract to flags
							dest := "/tmp/helmper-export/"
							plainHTTP := false
							tar := true
							// tarCollection := true // TODO: Implement this

							var (
								verbose bool = state.GetValue[bool](v, "verbose")
							)
							if verbose {
								slog.SetLogLoggerLevel(slog.LevelDebug)
							}

							// Read input artifact files
							data, err := helm.ReadYAMLFromFile("artifacts", rc)
							if err != nil {
								done <- err
							}

							for c, d := range data {
								// Pull Helm Chart
								p, err := c.PullTar(settings, dest)
								if err != nil {
									slog.Error("Failed to pull Helm chart", slog.Any("error", err))
									done <- err
									return
								}
								ie := importEntry{
									Name:     "charts/" + c.Name,
									Tag:      c.Version,
									FilePath: filepath.Base(p),
									Type:     "HELM CHART",
								}
								if !tar {
									slog.Info("Extracting Helm Chart from tar")

									// Ensure the destination directory exists
									destDir := filepath.Join(dest, c.Name)
									if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
										fmt.Printf("Failed to create destination directory: %v\n", err)
										return
									}

									chart, err := loader.Load(p)
									if err != nil {
										fmt.Printf("Failed to load chart: %v\n", err)
										return
									}

									// Extract the chart to the destination directory
									if err := chartutil.SaveDir(chart, destDir); err != nil {
										fmt.Printf("Failed to extract chart: %v\n", err)
										return
									}

									ie.FilePath = destDir

									if err = os.Remove(p); err != nil {
										fmt.Printf("Failed to remove tar file: %v\n", err)
										return
									}
								}
								index = append(index, &ie)

								// Pull Images
								for i := range d {

									ref, err := reference.ParseNamed(i.String())
									if err != nil {
										slog.Error("Failed to parse image reference", slog.Any("error", err))
										done <- err
									}

									slog.Info("Pulling image", slog.String("image", ref.Name()), slog.String("tag", i.Tag))
									_, err = registry.PullOCI(ctx, ref.Name(), i.Tag, dest, plainHTTP)
									if err != nil {
										slog.Error("Failed to pull image", slog.Any("error", err))
										done <- err
									}

									// Make ref.Name() filename friendly
									filenameFriendlyRefName := strings.ReplaceAll(ref.Name(), "/", "_")
									filenameFriendlyRefName = strings.ReplaceAll(filenameFriendlyRefName, ":", "_")
									p := filepath.Join(dest, filenameFriendlyRefName+"-"+i.Tag)

									// Construct tar
									if tar {
										tarFile, err := os.Create(p + ".tar")
										if err != nil {
											slog.Error("Failed to create tar file", slog.Any("error", err))
											done <- err
										}
										defer tarFile.Close()
										err = file.TarDirectory(ctx, p, "", tarFile, false, make([]byte, 32*1024))
										if err != nil {
											slog.Error("Failed to tar directory", slog.Any("error", err))
											done <- err
										}

										if err = os.RemoveAll(p); err != nil {
											fmt.Printf("Failed to remove folder: %v\n", err)
											return
										}
									}

									n, err := i.ImageName()
									if err != nil {
										slog.Error("Failed to get image name", slog.Any("error", err))
										done <- err
									}

									fp := func() string {
										b := filepath.Base(p)
										if tar {
											return b + ".tar"
										}
										return b
									}()
									ie := importEntry{
										Registry: i.Registry,
										Name:     n,
										Tag:      i.Tag,
										FilePath: fp,
										Type:     "OCI ARTIFACT",
									}
									index = append(index, &ie)
								}

							}

							// Marshal index to JSON
							indexJSON, err := json.MarshalIndent(index, "", "  ")
							if err != nil {
								slog.Error("Failed to marshal index to JSON", slog.Any("error", err))
								done <- err
								return
							}

							// Write JSON to file
							indexFilePath := filepath.Join(dest, "index.json")
							if err := os.WriteFile(indexFilePath, indexJSON, 0644); err != nil {
								slog.Error("Failed to write index to file", slog.Any("error", err))
								done <- err
								return
							}

							slog.Info("Index written to file", slog.String("file", indexFilePath))

							// Tar tars
							// if tarCollection {
							// 	// Construct tar
							// 	tarFile, err := os.Create(dest + ".tar")
							// 	if err != nil {
							// 		slog.Error("Failed to create tar file", slog.Any("error", err))
							// 		done <- err
							// 	}
							// 	defer tarFile.Close()
							// 	err = tarDirectory(ctx, dest, "", tarFile, false, make([]byte, 32*1024))
							// 	if err != nil {
							// 		slog.Error("Failed to tar directory", slog.Any("error", err))
							// 		done <- err
							// 	}
							// }

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
