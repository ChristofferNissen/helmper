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
	Data          helm.RegistryImageStatus                    // image data
	Data2         map[*registry.Registry]map[*helm.Chart]bool // chart data
	ChartData     helm.ChartData
	//RegistryImage map[*registry.Registry]map[*image.Image]bool
	RegistryImage    map[*helm.Chart]map[*image.Image][]bool
}

type ChartArtifact struct {
	ChartOverview string `json:"chart_overview"`
	ChartName     string `json:"chart_name"`
	Repository    string `json:"repository"`
	Registry      string `json:"registry"`
	ChartVersion  string `json:"chart_version"`
	Import        bool   `json:"import"`
}

type ImageArtifact struct {
	ImageOverview string `json:"image_overview"`
	ImageName     string `json:"image_name"`
	ImageTag      string `json:"image_tag"`
	Repository    string `json:"repository"`
	Import        bool   `json:"import"`
}

func (eo *ExportOption) Run(ctx context.Context) ([]string, []string, error) {
	imgOverview := []string{}
	chartOverview := []string{}
	chartDataOverview := []string{}

	// Collect image data
	for reg, imgs := range eo.Data {
		for img := range imgs {
			overview := fmt.Sprintf("Registry: %s, Image: %s, Repository: %s, Tag: %s",
				reg.GetName(),
				img.String(), img.Repository, img.Tag)
			fmt.Println("Registry - data 1:", reg)
			fmt.Println("Image - data 1:", imgs)
			imgOverview = append(imgOverview, overview)
		}
	}
	// Collect chart data
	for reg, charts := range eo.Data2 {
		for chart := range charts {
			overview := fmt.Sprintf("Registry: %s, Chart: %s, Version: %s",
				reg.Name, chart.Name, chart.Version)
				fmt.Println("Registry - data 2:", reg)
				fmt.Println("Chart - data 2:", chart)
				chartPath := fmt.Sprintf("charts/%s", chart.Name)
				fmt.Println(chartPath) 
			chartOverview = append(chartOverview, overview)
		}
	}

	// Collect chart data (2)
	for reg, charts := range eo.RegistryImage {
		// for chart := range charts {
			overview := fmt.Sprintf("Reg: %s, Charts: %s",
				reg, charts)
			chartDataOverview = append(chartDataOverview, overview)
		// }
	}

	// Convert to JSON (IMAGES) ("Registry: %s, Image: %s, Repository: %s, Tag: %s"
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
	// Convert to JSON (CHARTS) ("Registry: %s, Chart: %s, Version: %s"
	for _, overview := range chartOverview {
		ca := ChartArtifact{
			ChartOverview: overview,
			Registry:      strings.Split(overview, ", ")[0],
			ChartName:     strings.Split(overview, ", ")[1],
			ChartVersion:  strings.Split(overview, ", ")[2],
		}
		chartArtifacts = append(chartArtifacts, ca)
	}

	chartData := []ChartArtifact{}
	// Convert to JSON (CHARTS) ("Registry: %s, Chart: %s, Version: %s"
	for _, overview := range chartDataOverview {
		cd := ChartArtifact{
			ChartOverview: overview,
		}
		chartData = append(chartData, cd)
	}

	exportData := struct {
		Images []ImageArtifact `json:"images"`
		Charts []ChartArtifact `json:"charts"`
		Data   []ChartArtifact `json:"chart_data"`
	}{
		Images: artifacts,
		Charts: chartArtifacts,
		Data:   chartData,
	}

	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	// Write JSON to file
	err = os.WriteFile("artifacts.json", jsonData, 0644)
	if err != nil {
		return nil, nil, err
	}

	slog.Info("Exported artifacts to artifacts.json")
	return imgOverview, chartOverview, nil
}
