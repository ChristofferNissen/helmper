package exportArtifacts

import (
	"context"
	"os"
	"testing"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
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

    mockData2 := map[*registry.Registry]map[*helm.Chart]bool{
        mockRegistry: {
            mockChart: true,
        },
    }

    eo := &ExportOption{
        Image:  mockData,
        Chart: mockData2,
    }

    // Act
    imgOverview, chartOverview, err := eo.Run(context.Background())

    // Assert
    assert.NoError(t, err, "expected no error during execution")
    assert.FileExists(t, "artifacts.json", "expected file to be created")
    assert.NotEmpty(t, imgOverview, "expected image overview to be populated")
    assert.NotEmpty(t, chartOverview, "expected chart overview to be populated")

    // Cleanup
    os.Remove("artifacts.json")
}