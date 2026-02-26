package exportArtifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/spf13/afero"
)

// Report is the top-level JSON structure for the repository report.
type Report struct {
	Registries []RegistryReport `json:"registries"`
}

// RegistryReport represents a single target registry and its expected repositories.
type RegistryReport struct {
	Name              string           `json:"name"`
	URL               string           `json:"url"`
	ChartRepositories []ChartRepoEntry `json:"chart_repositories"`
	ImageRepositories []ImageRepoEntry `json:"image_repositories"`
}

// ChartRepoEntry represents a chart repository in the target registry.
type ChartRepoEntry struct {
	Repository string `json:"repository"`
	Chart      string `json:"chart"`
	Version    string `json:"version"`
}

// ImageRepoEntry represents an image repository in the target registry.
type ImageRepoEntry struct {
	Repository  string `json:"repository"`
	SourceImage string `json:"source_image"`
	Tag         string `json:"tag"`
}

// ReportOption configures the repository report generation.
type ReportOption struct {
	Fs    afero.Fs
	Chart helm.RegistryChartStatus
	Image helm.RegistryImageStatus
}

// Run generates the JSON report organized by registry and writes it to the specified folder.
func (ro *ReportOption) Run(_ context.Context, folder string) (*Report, error) {
	// Collect unique registries from both maps
	registrySet := make(map[string]*registry.Registry)
	registryOrder := make([]*registry.Registry, 0)

	for r := range ro.Chart {
		if _, exists := registrySet[r.URL]; !exists {
			registrySet[r.URL] = r
			registryOrder = append(registryOrder, r)
		}
	}
	for r := range ro.Image {
		if _, exists := registrySet[r.URL]; !exists {
			registrySet[r.URL] = r
			registryOrder = append(registryOrder, r)
		}
	}

	// Sort by URL for deterministic output
	sort.Slice(registryOrder, func(i, j int) bool {
		return registryOrder[i].URL < registryOrder[j].URL
	})

	report := Report{
		Registries: make([]RegistryReport, 0, len(registryOrder)),
	}

	for _, r := range registryOrder {
		rr := RegistryReport{
			Name:              r.GetName(),
			URL:               r.URL,
			ChartRepositories: make([]ChartRepoEntry, 0),
			ImageRepositories: make([]ImageRepoEntry, 0),
		}

		// Charts
		if chartMap, ok := ro.Chart[r]; ok {
			for c := range chartMap {
				if c.Name == "images" {
					continue
				}
				rr.ChartRepositories = append(rr.ChartRepositories, ChartRepoEntry{
					Repository: fmt.Sprintf("charts/%s", c.Name),
					Chart:      c.Name,
					Version:    c.Version,
				})
			}
		}
		sort.Slice(rr.ChartRepositories, func(i, j int) bool {
			if rr.ChartRepositories[i].Chart == rr.ChartRepositories[j].Chart {
				return rr.ChartRepositories[i].Version < rr.ChartRepositories[j].Version
			}
			return rr.ChartRepositories[i].Chart < rr.ChartRepositories[j].Chart
		})

		// Images
		if imgMap, ok := ro.Image[r]; ok {
			for img := range imgMap {
				name, err := img.ImageName()
				if err != nil {
					slog.Warn("skipping image with unparseable name",
						slog.String("image", img.String()),
						slog.String("error", err.Error()))
					continue
				}

				if r.PrefixSource {
					prefixed, err := image.UpdateNameWithPrefixSource(img)
					if err != nil {
						slog.Warn("skipping image; could not compute prefixed name",
							slog.String("image", img.String()),
							slog.String("error", err.Error()))
						continue
					}
					name = prefixed
				}

				tag := img.Tag
				if img.UseDigest && img.Tag == "" {
					tag = img.Digest
				}

				rr.ImageRepositories = append(rr.ImageRepositories, ImageRepoEntry{
					Repository:  name,
					SourceImage: img.String(),
					Tag:         tag,
				})
			}
		}
		sort.Slice(rr.ImageRepositories, func(i, j int) bool {
			if rr.ImageRepositories[i].Repository == rr.ImageRepositories[j].Repository {
				return rr.ImageRepositories[i].Tag < rr.ImageRepositories[j].Tag
			}
			return rr.ImageRepositories[i].Repository < rr.ImageRepositories[j].Repository
		})

		report.Registries = append(report.Registries, rr)
	}

	// Serialize
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		slog.Error("Failed to generate report JSON", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to generate report JSON: %w", err)
	}

	// Write file
	destPath := "report.json"
	if folder != "" {
		err = ro.Fs.MkdirAll(folder, 0755)
		if err != nil {
			slog.Error("Failed to create directory", slog.String("folder", folder), slog.String("error", err.Error()))
			return nil, fmt.Errorf("failed to create directory %s: %w", folder, err)
		}
		destPath = fmt.Sprintf("%s/%s", folder, destPath)
	} else {
		destPath = "./" + destPath
		slog.Info("No folder specified, saving report in the working directory")
	}

	err = afero.WriteFile(ro.Fs, destPath, jsonData, 0644)
	if err != nil {
		slog.Error("Failed to write report", slog.String("path", destPath), slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to write report to %s: %w", destPath, err)
	}

	slog.Info("Exported repository report", slog.String("path", destPath))
	return &report, nil
}
