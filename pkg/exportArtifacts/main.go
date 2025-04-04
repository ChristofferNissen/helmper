package exportArtifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
)

type ExportOption struct {
	Image helm.RegistryImageStatus   
	Chart helm.RegistryChartStatus                 // image data
	Data2     map[*registry.Registry]map[*helm.Chart]bool // chart data
	ChartData helm.ChartData
	RegistryImage map[*helm.Chart]map[*image.Image][]bool
	Registries    []*registry.Registry
}

type ChartArtifact struct {
	ChartOverview string `json:"chart_overview"`
	ChartName     string `json:"chart_name"`
	Repository    string `json:"repository"`
	ChartVersion  string `json:"chart_version"`
	ChartPath 	  string `json:"chart_artifact_path"`
}

type ImageArtifact struct {
	ImageOverview string `json:"image_overview"`
	ImageName     string `json:"image_name"`
	ImageTag      string `json:"image_tag"`
	Repository    string `json:"repository"`
}

func (eo *ExportOption) Run(ctx context.Context) ([]string, []string, error) {
	imgOverview := []string{}
	chartOverview := []string{}

	// Collect image data
	for r, i := range eo.Image {
		for img := range i {
			overview := fmt.Sprintf("Registry: %s, Image: %s, Repository: %s, Tag: %s",
				r.GetName(),
				img.String(), img.Repository, img.Tag)
			imgOverview = append(imgOverview, overview)
		}
	}
	// Collect chart data
	for r, c := range eo.Chart {
		for chart := range c {
			overview := fmt.Sprintf("Registry: %s, Chart: %s, Version: %s, ChartPath: %s",
				r.Name, chart.Name, chart.Version, fmt.Sprintf("charts/%s", chart.Name))
			chartOverview = append(chartOverview, overview)
		}
	}
    // Convert to JSON
	artifacts := []ImageArtifact{}
	for _, overview := range imgOverview {
		ia := ImageArtifact{
			ImageOverview: overview,
			ImageName:     strings.Split(overview, ", ")[1],
			Repository:    strings.Split(overview, ", ")[2],
			ImageTag:      strings.Split(overview, ", ")[3],
		}
		artifacts = append(artifacts, ia)
	}

	chartArtifacts := []ChartArtifact{}
	for _, overview := range chartOverview {
		ca := ChartArtifact{
			ChartOverview: overview,
			ChartName:     strings.Split(overview, ", ")[1],
			ChartVersion:  strings.Split(overview, ", ")[2],
			ChartPath:     strings.Split(overview, ", ")[3],
		}
		chartArtifacts = append(chartArtifacts, ca)
	}

	exportData := struct {
		Images []ImageArtifact `json:"images"`
		Charts []ChartArtifact `json:"charts"`
	}{
		Images: artifacts,
		Charts: chartArtifacts,
	}

	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	err = os.WriteFile("artifacts.json", jsonData, 0644)
	if err != nil {
		return nil, nil, err
	}

	slog.Info("Exported artifacts to artifacts.json")
	return imgOverview, chartOverview, nil
}