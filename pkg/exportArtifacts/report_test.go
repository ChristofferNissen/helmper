package exportArtifacts

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportOption_Basic(t *testing.T) {
	mockRegistry := &registry.Registry{
		Name: "my-ecr",
		URL:  "oci://123456789.dkr.ecr.us-east-1.amazonaws.com",
	}
	mockImage := &image.Image{
		Registry:   "docker.io",
		Repository: "library/nginx",
		Tag:        "1.25",
	}
	mockChart := &helm.Chart{Name: "kyverno", Version: "3.1.1"}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs: mockFs,
		Chart: helm.RegistryChartStatus{
			mockRegistry: {mockChart: true},
		},
		Image: helm.RegistryImageStatus{
			mockRegistry: {mockImage: true},
		},
	}

	report, err := ro.Run(context.Background(), "")
	require.NoError(t, err)

	assert.Len(t, report.Registries, 1)
	assert.Equal(t, "my-ecr", report.Registries[0].Name)
	assert.Equal(t, "oci://123456789.dkr.ecr.us-east-1.amazonaws.com", report.Registries[0].URL)

	// Charts
	assert.Len(t, report.Registries[0].ChartRepositories, 1)
	assert.Equal(t, "charts/kyverno", report.Registries[0].ChartRepositories[0].Repository)
	assert.Equal(t, "kyverno", report.Registries[0].ChartRepositories[0].Chart)
	assert.Equal(t, "3.1.1", report.Registries[0].ChartRepositories[0].Version)

	// Images
	assert.Len(t, report.Registries[0].ImageRepositories, 1)
	assert.Equal(t, "library/nginx", report.Registries[0].ImageRepositories[0].Repository)
	assert.Equal(t, "docker.io/library/nginx:1.25", report.Registries[0].ImageRepositories[0].SourceImage)
	assert.Equal(t, "1.25", report.Registries[0].ImageRepositories[0].Tag)

	// Verify file was written and is valid JSON
	content, err := afero.ReadFile(mockFs, "report.json")
	require.NoError(t, err)

	var parsed Report
	err = json.Unmarshal(content, &parsed)
	assert.NoError(t, err)
	assert.EqualValues(t, *report, parsed)
}

func TestReportOption_PrefixSource(t *testing.T) {
	mockRegistry := &registry.Registry{
		Name:         "my-ecr",
		URL:          "oci://123456789.dkr.ecr.us-east-1.amazonaws.com",
		PrefixSource: true,
	}
	mockImage := &image.Image{
		Registry:   "docker.io",
		Repository: "library/nginx",
		Tag:        "1.25",
	}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs:    mockFs,
		Chart: helm.RegistryChartStatus{},
		Image: helm.RegistryImageStatus{
			mockRegistry: {mockImage: true},
		},
	}

	report, err := ro.Run(context.Background(), "")
	require.NoError(t, err)

	assert.Len(t, report.Registries, 1)
	assert.Len(t, report.Registries[0].ImageRepositories, 1)
	assert.Equal(t, "docker/library/nginx", report.Registries[0].ImageRepositories[0].Repository)
}

func TestReportOption_MultipleRegistries(t *testing.T) {
	reg1 := &registry.Registry{
		Name: "plain",
		URL:  "oci://aaa.ecr.amazonaws.com",
	}
	reg2 := &registry.Registry{
		Name:         "prefixed",
		URL:          "oci://bbb.ecr.amazonaws.com",
		PrefixSource: true,
	}
	mockImage := &image.Image{
		Registry:   "ghcr.io",
		Repository: "kyverno/kyverno",
		Tag:        "v1.11.1",
	}
	mockChart := &helm.Chart{Name: "kyverno", Version: "3.1.1"}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs: mockFs,
		Chart: helm.RegistryChartStatus{
			reg1: {mockChart: true},
			reg2: {mockChart: true},
		},
		Image: helm.RegistryImageStatus{
			reg1: {mockImage: true},
			reg2: {mockImage: true},
		},
	}

	report, err := ro.Run(context.Background(), "")
	require.NoError(t, err)

	assert.Len(t, report.Registries, 2)

	// Sorted by URL, so aaa comes first
	assert.Equal(t, "plain", report.Registries[0].Name)
	assert.Equal(t, "kyverno/kyverno", report.Registries[0].ImageRepositories[0].Repository)

	assert.Equal(t, "prefixed", report.Registries[1].Name)
	assert.Equal(t, "ghcr/kyverno/kyverno", report.Registries[1].ImageRepositories[0].Repository)
}

func TestReportOption_EmptyData(t *testing.T) {
	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs:    mockFs,
		Chart: helm.RegistryChartStatus{},
		Image: helm.RegistryImageStatus{},
	}

	report, err := ro.Run(context.Background(), "")
	require.NoError(t, err)

	assert.Empty(t, report.Registries)
}

func TestReportOption_SkipsImagesChart(t *testing.T) {
	mockRegistry := &registry.Registry{
		Name: "my-ecr",
		URL:  "oci://123456789.dkr.ecr.us-east-1.amazonaws.com",
	}
	// The "images" chart is a synthetic placeholder used internally
	imagesChart := &helm.Chart{Name: "images", Version: "0.0.0"}
	realChart := &helm.Chart{Name: "kyverno", Version: "3.1.1"}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs: mockFs,
		Chart: helm.RegistryChartStatus{
			mockRegistry: {
				imagesChart: true,
				realChart:   true,
			},
		},
		Image: helm.RegistryImageStatus{},
	}

	report, err := ro.Run(context.Background(), "")
	require.NoError(t, err)

	assert.Len(t, report.Registries[0].ChartRepositories, 1)
	assert.Equal(t, "kyverno", report.Registries[0].ChartRepositories[0].Chart)
}

func TestReportOption_CustomFolder(t *testing.T) {
	mockRegistry := &registry.Registry{
		Name: "my-ecr",
		URL:  "oci://123456789.dkr.ecr.us-east-1.amazonaws.com",
	}
	mockChart := &helm.Chart{Name: "kyverno", Version: "3.1.1"}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs: mockFs,
		Chart: helm.RegistryChartStatus{
			mockRegistry: {mockChart: true},
		},
		Image: helm.RegistryImageStatus{},
	}

	_, err := ro.Run(context.Background(), "/output/report")
	require.NoError(t, err)

	exists, err := afero.Exists(mockFs, "/output/report/report.json")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestReportOption_DigestOnlyImage(t *testing.T) {
	mockRegistry := &registry.Registry{
		Name: "my-ecr",
		URL:  "oci://123456789.dkr.ecr.us-east-1.amazonaws.com",
	}
	// Use a valid 64-char hex SHA256 digest
	digest := "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	mockImage := &image.Image{
		Registry:   "docker.io",
		Repository: "library/nginx",
		Tag:        "",
		Digest:     digest,
		UseDigest:  true,
	}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs:    mockFs,
		Chart: helm.RegistryChartStatus{},
		Image: helm.RegistryImageStatus{
			mockRegistry: {mockImage: true},
		},
	}

	report, err := ro.Run(context.Background(), "")
	require.NoError(t, err)

	require.Len(t, report.Registries[0].ImageRepositories, 1)
	assert.Equal(t, "library/nginx", report.Registries[0].ImageRepositories[0].Repository)
	assert.Equal(t, digest, report.Registries[0].ImageRepositories[0].Tag)
}

func TestReportOption_IncludesItemsNotMarkedForImport(t *testing.T) {
	// The report should list ALL resources regardless of the import boolean.
	// Items with bool=false (already exist in registry) must still appear.
	mockRegistry := &registry.Registry{
		Name: "my-ecr",
		URL:  "oci://123456789.dkr.ecr.us-east-1.amazonaws.com",
	}
	existingChart := &helm.Chart{Name: "prometheus", Version: "25.8.0"}
	newChart := &helm.Chart{Name: "kyverno", Version: "3.1.1"}
	existingImage := &image.Image{
		Registry:   "docker.io",
		Repository: "library/nginx",
		Tag:        "1.25",
	}
	newImage := &image.Image{
		Registry:   "ghcr.io",
		Repository: "kyverno/kyverno",
		Tag:        "v1.11.1",
	}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs: mockFs,
		Chart: helm.RegistryChartStatus{
			mockRegistry: {
				existingChart: false, // already in registry
				newChart:      true,  // needs import
			},
		},
		Image: helm.RegistryImageStatus{
			mockRegistry: {
				existingImage: false, // already in registry
				newImage:      true,  // needs import
			},
		},
	}

	report, err := ro.Run(context.Background(), "")
	require.NoError(t, err)

	assert.Len(t, report.Registries[0].ChartRepositories, 2)
	assert.Len(t, report.Registries[0].ImageRepositories, 2)
}

func TestReportOption_DeterministicSortOrder(t *testing.T) {
	mockRegistry := &registry.Registry{
		Name: "my-ecr",
		URL:  "oci://123456789.dkr.ecr.us-east-1.amazonaws.com",
	}
	chartA := &helm.Chart{Name: "argo-cd", Version: "5.51.4"}
	chartB := &helm.Chart{Name: "kyverno", Version: "3.1.1"}
	chartC := &helm.Chart{Name: "prometheus", Version: "25.8.0"}

	imgA := &image.Image{Registry: "docker.io", Repository: "library/nginx", Tag: "1.25"}
	imgB := &image.Image{Registry: "ghcr.io", Repository: "kyverno/kyverno", Tag: "v1.11.1"}
	imgC := &image.Image{Registry: "quay.io", Repository: "argoproj/argocd", Tag: "v2.9.3"}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs: mockFs,
		Chart: helm.RegistryChartStatus{
			mockRegistry: {chartC: true, chartA: true, chartB: true},
		},
		Image: helm.RegistryImageStatus{
			mockRegistry: {imgC: true, imgA: true, imgB: true},
		},
	}

	// Run multiple times and verify output is always the same
	var firstJSON []byte
	for i := 0; i < 5; i++ {
		report, err := ro.Run(context.Background(), "")
		require.NoError(t, err)

		jsonData, err := json.MarshalIndent(report, "", "  ")
		require.NoError(t, err)

		if i == 0 {
			firstJSON = jsonData

			// Verify sort order: charts sorted by name
			assert.Equal(t, "argo-cd", report.Registries[0].ChartRepositories[0].Chart)
			assert.Equal(t, "kyverno", report.Registries[0].ChartRepositories[1].Chart)
			assert.Equal(t, "prometheus", report.Registries[0].ChartRepositories[2].Chart)

			// Verify sort order: images sorted by repository
			assert.Equal(t, "argoproj/argocd", report.Registries[0].ImageRepositories[0].Repository)
			assert.Equal(t, "kyverno/kyverno", report.Registries[0].ImageRepositories[1].Repository)
			assert.Equal(t, "library/nginx", report.Registries[0].ImageRepositories[2].Repository)
		} else {
			assert.JSONEq(t, string(firstJSON), string(jsonData), "output should be deterministic across runs")
		}
	}
}

func TestReportOption_RegistryOnlyInImageMap(t *testing.T) {
	// Registry appears only in the Image map, not in Chart map.
	imgRegistry := &registry.Registry{
		Name: "images-only",
		URL:  "oci://images.ecr.amazonaws.com",
	}
	mockImage := &image.Image{
		Registry:   "docker.io",
		Repository: "library/redis",
		Tag:        "7.2",
	}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs:    mockFs,
		Chart: helm.RegistryChartStatus{},
		Image: helm.RegistryImageStatus{
			imgRegistry: {mockImage: true},
		},
	}

	report, err := ro.Run(context.Background(), "")
	require.NoError(t, err)

	require.Len(t, report.Registries, 1)
	assert.Equal(t, "images-only", report.Registries[0].Name)
	assert.Empty(t, report.Registries[0].ChartRepositories)
	assert.Len(t, report.Registries[0].ImageRepositories, 1)
}

func TestReportOption_PrefixSourceMultipleSources(t *testing.T) {
	// Different source registries produce different prefixes.
	mockRegistry := &registry.Registry{
		Name:         "prefixed",
		URL:          "oci://123456789.dkr.ecr.us-east-1.amazonaws.com",
		PrefixSource: true,
	}
	imgDocker := &image.Image{
		Registry:   "docker.io",
		Repository: "library/nginx",
		Tag:        "1.25",
	}
	imgGhcr := &image.Image{
		Registry:   "ghcr.io",
		Repository: "kyverno/kyverno",
		Tag:        "v1.11.1",
	}
	imgQuay := &image.Image{
		Registry:   "quay.io",
		Repository: "argoproj/argocd",
		Tag:        "v2.9.3",
	}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs:    mockFs,
		Chart: helm.RegistryChartStatus{},
		Image: helm.RegistryImageStatus{
			mockRegistry: {
				imgDocker: true,
				imgGhcr:   true,
				imgQuay:   true,
			},
		},
	}

	report, err := ro.Run(context.Background(), "")
	require.NoError(t, err)

	repos := report.Registries[0].ImageRepositories
	require.Len(t, repos, 3)

	// Sorted by repository name
	repoNames := make([]string, len(repos))
	for i, r := range repos {
		repoNames[i] = r.Repository
	}
	assert.Equal(t, []string{
		"docker/library/nginx",
		"ghcr/kyverno/kyverno",
		"quay/argoproj/argocd",
	}, repoNames)
}

func TestReportOption_JSONFieldNames(t *testing.T) {
	// Verify the JSON serialization uses the expected field names.
	mockRegistry := &registry.Registry{
		Name: "my-ecr",
		URL:  "oci://ecr.amazonaws.com",
	}
	mockChart := &helm.Chart{Name: "nginx", Version: "1.0.0"}
	mockImage := &image.Image{
		Registry:   "docker.io",
		Repository: "library/nginx",
		Tag:        "latest",
	}

	mockFs := afero.NewMemMapFs()
	ro := &ReportOption{
		Fs: mockFs,
		Chart: helm.RegistryChartStatus{
			mockRegistry: {mockChart: true},
		},
		Image: helm.RegistryImageStatus{
			mockRegistry: {mockImage: true},
		},
	}

	_, err := ro.Run(context.Background(), "")
	require.NoError(t, err)

	content, err := afero.ReadFile(mockFs, "report.json")
	require.NoError(t, err)

	// Parse as raw map to verify exact JSON field names
	var raw map[string]any
	err = json.Unmarshal(content, &raw)
	require.NoError(t, err)

	// Top level
	assert.Contains(t, raw, "registries")

	registries := raw["registries"].([]any)
	require.Len(t, registries, 1)
	reg := registries[0].(map[string]any)

	// Registry fields
	assert.Contains(t, reg, "name")
	assert.Contains(t, reg, "url")
	assert.Contains(t, reg, "chart_repositories")
	assert.Contains(t, reg, "image_repositories")

	// Chart entry fields
	charts := reg["chart_repositories"].([]any)
	require.Len(t, charts, 1)
	chart := charts[0].(map[string]any)
	assert.Contains(t, chart, "repository")
	assert.Contains(t, chart, "chart")
	assert.Contains(t, chart, "version")

	// Image entry fields
	images := reg["image_repositories"].([]any)
	require.Len(t, images, 1)
	img := images[0].(map[string]any)
	assert.Contains(t, img, "repository")
	assert.Contains(t, img, "source_image")
	assert.Contains(t, img, "tag")
}
