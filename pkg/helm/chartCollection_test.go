package helm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

// setupTestSettings creates a temporary directory with a valid helm
// RepositoryConfig so that SetupHelm's updateRepositories call succeeds.
// The caller must call the returned cleanup function when done.
func setupTestSettings(t *testing.T) (*cli.EnvSettings, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "helmper-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	settings := cli.New()
	settings.RepositoryCache = tempDir

	f := repo.NewFile()
	repoFile := filepath.Join(tempDir, "repositories.yaml")
	if err := f.WriteFile(repoFile, 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to write repo file: %v", err)
	}
	settings.RepositoryConfig = repoFile

	cleanup := func() {
		os.RemoveAll(tempDir)
	}
	return settings, cleanup
}

func TestSetupHelm_NonExistentExactVersion(t *testing.T) {
	settings, cleanup := setupTestSettings(t)
	defer cleanup()

	mockClient := new(MockRegistryClient)
	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0", "2.0.0"}, nil)

	// Version "99.99.99" is a valid semver range (exact match) but no tag matches it.
	// ResolveVersions will return an empty list without error, and the chart
	// should NOT be silently dropped.
	collection := ChartCollection{
		Charts: []*Chart{
			{
				Name:    "my-chart",
				Version: "99.99.99",
				Repo: repo.Entry{
					URL: "oci://registry.example.com/charts/my-chart",
				},
				PlainHTTP:      true,
				RegistryClient: mockClient,
			},
		},
	}

	_, err := collection.SetupHelm(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "my-chart:99.99.99")
}

func TestSetupHelm_NonExistentGlobVersion(t *testing.T) {
	settings, cleanup := setupTestSettings(t)
	defer cleanup()

	mockClient := new(MockRegistryClient)
	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0"}, nil)

	// Version "9.0.*" is not a valid semver range, so ResolveVersions fails.
	// ResolveVersion converts it to "9.0.x" (>=9.0.0, <9.1.0), but no tags match.
	collection := ChartCollection{
		Charts: []*Chart{
			{
				Name:    "my-chart",
				Version: "9.0.*",
				Repo: repo.Entry{
					URL: "oci://registry.example.com/charts/my-chart",
				},
				PlainHTTP:      true,
				RegistryClient: mockClient,
			},
		},
	}

	_, err := collection.SetupHelm(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "my-chart:9.0.*")
}

func TestSetupHelm_MixedValidAndInvalidVersions(t *testing.T) {
	settings, cleanup := setupTestSettings(t)
	defer cleanup()

	mockClient := new(MockRegistryClient)
	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0", "2.0.0"}, nil)

	// One chart has a valid version range, the other doesn't exist.
	// The whole run should fail because of the invalid chart.
	collection := ChartCollection{
		Charts: []*Chart{
			{
				Name:    "good-chart",
				Version: ">= 1.0.0",
				Repo: repo.Entry{
					URL: "oci://registry.example.com/charts/good-chart",
				},
				PlainHTTP:      true,
				RegistryClient: mockClient,
			},
			{
				Name:    "bad-chart",
				Version: "99.99.99",
				Repo: repo.Entry{
					URL: "oci://registry.example.com/charts/bad-chart",
				},
				PlainHTTP:      true,
				RegistryClient: mockClient,
			},
		},
	}

	_, err := collection.SetupHelm(settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad-chart:99.99.99")
	// The good chart should not appear in the error
	assert.NotContains(t, err.Error(), "good-chart")
}

func TestSetupHelm_AllChartsSkippedReportsAll(t *testing.T) {
	settings, cleanup := setupTestSettings(t)
	defer cleanup()

	mockClient := new(MockRegistryClient)
	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0"}, nil)

	collection := ChartCollection{
		Charts: []*Chart{
			{
				Name:    "chart-a",
				Version: "99.99.99",
				Repo: repo.Entry{
					URL: "oci://registry.example.com/charts/chart-a",
				},
				PlainHTTP:      true,
				RegistryClient: mockClient,
			},
			{
				Name:    "chart-b",
				Version: "88.88.88",
				Repo: repo.Entry{
					URL: "oci://registry.example.com/charts/chart-b",
				},
				PlainHTTP:      true,
				RegistryClient: mockClient,
			},
		},
	}

	_, err := collection.SetupHelm(settings)
	assert.Error(t, err)

	errMsg := err.Error()
	assert.True(t, strings.Contains(errMsg, "chart-a:99.99.99"), "error should mention chart-a")
	assert.True(t, strings.Contains(errMsg, "chart-b:88.88.88"), "error should mention chart-b")
}
