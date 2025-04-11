package exportArtifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/spf13/afero"
)

type ExportOption struct {
	Fs    afero.Fs
	Image helm.RegistryImageStatus
	Chart helm.RegistryChartStatus
}

type ChartArtifact struct {
	ChartOverview string `json:"chart_overview"`
	ChartName     string `json:"chart_name"`
	Repository    string `json:"repository"`
	ChartVersion  string `json:"chart_version"`
	ChartPath     string `json:"chart_artifact_path"`
}

type ImageArtifact struct {
	ImageOverview string `json:"image_overview"`
	ImageName     string `json:"image_name"`
	ImageTag      string `json:"image_tag"`
}

func (eo *ExportOption) Run(ctx context.Context, folder string) ([]ImageArtifact, []ChartArtifact, error) {
	// Collect image data
	imageArtifacts := []ImageArtifact{}
	for r, i := range eo.Image {
		for img := range i {
			overview := fmt.Sprintf("Registry: %s, Image: %s, Tag: %s",
				r.GetName(),
				img.String(), img.Tag)
				ia := ImageArtifact{
					ImageOverview: overview,
					ImageName:     img.String(),
					ImageTag:     img.Tag,
				}
			imageArtifacts = append(imageArtifacts, ia)
		}
	}

	// Collect chart data
	chartArtifacts := []ChartArtifact{}
	for r, c := range eo.Chart {
		for chart := range c {
			overview := fmt.Sprintf("Registry: %s, Chart: %s, Version: %s, ChartPath: %s",
				r.Name, chart.Name, chart.Version, fmt.Sprintf("charts/%s", chart.Name))

			ca := ChartArtifact{
				ChartOverview: overview,
				ChartName:     chart.Name,
				ChartVersion:  chart.Version,
				ChartPath:     fmt.Sprintf("charts/%s", chart.Name),
			}
			chartArtifacts = append(chartArtifacts, ca)
		}
	}

	exportData := struct {
		Images []ImageArtifact `json:"images"`
		Charts []ChartArtifact `json:"charts"`
	}{
		Images: imageArtifacts,
		Charts: chartArtifacts,
	}

	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
        slog.Error("Failed to export data to JSON", slog.String("error", err.Error()))
        return nil, nil, fmt.Errorf("failed to export data to JSON: %w", err)
    }
	
	destPath := "artifacts.json"
	if folder != "" {
		err = eo.Fs.MkdirAll(folder, 0755)
		if err != nil {
			slog.Error("Failed to create directory", slog.String("folder", folder), slog.String("error", err.Error()))
			return nil, nil, fmt.Errorf("failed to save file in the specified location %s: %w", folder, err)
		}
		destPath = fmt.Sprintf("%s/%s", folder, destPath)
	} else {
		destPath = "./" + destPath
		slog.Info("No folder specified, saving in the root directory")
	}
	
	err = afero.WriteFile(eo.Fs, destPath, jsonData, 0644)
    if err != nil {
        slog.Error("Failed to write artifacts to", destPath, slog.String("error", err.Error()))
        return nil, nil, fmt.Errorf("failed to write artifacts to %s: %w", destPath, err)
    }

	slog.Info("Exported artifacts", slog.String("path", destPath))
	return imageArtifacts, chartArtifacts, nil
}