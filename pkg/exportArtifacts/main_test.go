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
)

func TestExportOptionRun(t *testing.T) {
	// Arrange
	mockRegistry := &registry.Registry{Name: "azure.registry.io"}
	mockImage := &image.Image{Repository: "argocd", Tag: "v2.0.5"}
	mockChart := &helm.Chart{Name: "prometheus", Version: "1.0.0"}

	mockData := helm.RegistryImageStatus{
		mockRegistry: {
			mockImage: true,
		},
	}

	mockData2 := helm.RegistryChartStatus {
		mockRegistry: {
			mockChart: true,
		},
	}

	mockFs := afero.NewMemMapFs() 
	eo := &ExportOption{
		Fs:    mockFs,
		Image: mockData,
		Chart: mockData2,
	}

	// Act
	imgOverview, chartOverview, err := eo.Run(context.Background(), "")

	// Assert	

    assert.NoError(t, err)

    content, err := afero.ReadFile(mockFs, "artifacts.json")
	assert.NoError(t, err)


	var artifact struct {
		Images []ImageArtifact `json:"images"`
		Charts []ChartArtifact `json:"charts"`
	}
	err = json.Unmarshal(content, &artifact)
	assert.NoError(t, err)

	assert.EqualValues(t, imgOverview, artifact.Images)
	assert.EqualValues(t, chartOverview, artifact.Charts)
}

func TestExportOptionRun_NoData(t *testing.T) {
    // Arrange
    mockFs := afero.NewMemMapFs()

    eo := &ExportOption{
        Fs:    mockFs,
        Image: helm.RegistryImageStatus{}, 
        Chart: helm.RegistryChartStatus{},
    }

    // Act
    imgOverview, chartOverview, err := eo.Run(context.Background(), "")

    // Assert
    assert.NoError(t, err)
    assert.Empty(t, imgOverview, "expected no image artifacts")
    assert.Empty(t, chartOverview, "expected no chart artifacts")
}