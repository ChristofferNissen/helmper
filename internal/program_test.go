package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

// "Integration" tests below. Tests the expected result of parsing a Helm Chart (number of charts, images)
//
// Chart counts are asserted exactly (determined by the pinned dependency tree).
// Image counts use a minimum bound (>= minExpectedImageCount) because helmper
// validates image availability against remote registries, and upstream registries
// may prune old tags over time (e.g. docker.io/bitnami/kubectl).

func createTempDir() (string, func(), error) {
	// Create a new temporary directory
	tempDir, err := os.MkdirTemp("", "tempdir_*")
	if err != nil {
		return "", nil, err
	}

	// Define the cleanup function
	cleanup := func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			fmt.Printf("Failed to remove temp dir: %v\n", err)
		} else {
			fmt.Printf("Temp dir %s removed.\n", tempDir)
		}
	}

	return tempDir, cleanup, nil
}

func testSettings() (*cli.EnvSettings, error) {
	// Create a temporary directory
	tempDir, cleanup, err := createTempDir()
	if err != nil {
		fmt.Printf("Error creating temp dir: %v\n", err)
		return nil, err
	}
	// Ensure cleanup is called to remove the temp directory
	defer cleanup()
	// Use the temp directory for your operations
	fmt.Printf("Temporary directory created: %s\n", tempDir)
	settings := cli.New()
	settings.RepositoryCache = tempDir
	f := repo.NewFile()
	repoFile := filepath.Join(tempDir, "repositories.yaml")
	f.WriteFile(repoFile, 0644)
	settings.RepositoryConfig = repoFile

	return settings, nil
}

// assertChartData verifies chart and image counts from a ChartOption.Run() result.
//
// chartCount is asserted exactly (deterministic from pinned chart dependencies).
//
// minImageCount is a lower bound: the test passes if imageCount >= minImageCount.
// This is necessary because ChartOption.Run() validates image availability against
// remote registries, which is non-deterministic: upstream registries may prune old
// tags (e.g. docker.io/bitnami/kubectl) or rate-limit concurrent requests, causing
// the actual count to vary between runs. Minimum bounds should be set close to the
// known-good count to still catch parsing regressions.
func assertChartData(t *testing.T, data helm.ChartData, chartCount int, minImageCount int) {
	t.Helper()

	if len(data) != chartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", chartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount < minImageCount {
		t.Fatalf("want at least '%d' images, got '%d'\n", minImageCount, imageCount)
	}

	t.Logf("found '%d' images (minimum expected: '%d')", imageCount, minImageCount)
}

func TestFindImagesWithoutCharts(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	charts := helm.ChartCollection{
		Charts: []*helm.Chart{},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 0, 0)
}

func TestFindImagesInHelmChartsOnPrometheusChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "prometheus",
				Repo: repo.Entry{
					Name: "prometheus-community",
					URL:  "https://prometheus-community.github.io/helm-charts",
				},
				Version:        "25.8.0",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 5, 5)
}

func TestFindImagesInHelmChartsOnPromtailChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "promtail",
				Repo: repo.Entry{
					Name: "grafana",
					URL:  "https://grafana.github.io/helm-charts",
				},
				Version:        "6.15.3",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 1)
}

func TestFindImagesInHelmChartsOnLokiChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "loki",
				Repo: repo.Entry{
					Name: "grafana",
					URL:  "https://grafana.github.io/helm-charts",
				},
				Version:        "5.38.0",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 2, 4)
}

func TestFindImagesInHelmChartsOnMimirDistributedChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "mimir-distributed",
				Repo: repo.Entry{
					Name: "grafana",
					URL:  "https://grafana.github.io/helm-charts",
				},
				Version:        "5.1.3",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 3, 7)
}

func TestFindImagesInHelmChartsOnGrafanaChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "grafana",
				Repo: repo.Entry{
					Name: "grafana",
					URL:  "https://grafana.github.io/helm-charts",
				},
				Version:        "7.0.9",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 4)
}

func TestFindImagesInHelmChartsOnCiliumChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "cilium",
				Repo: repo.Entry{
					Name: "cilium",
					URL:  "https://helm.cilium.io/",
				},
				Version:        "1.14.4",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 4)
}

func TestFindImagesInHelmChartsOnCertManagerChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "cert-manager",
				Repo: repo.Entry{
					Name: "cert-manager",
					URL:  "https://charts.jetstack.io",
				},
				Version:        "1.13.2",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,

		Settings: settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 4)
}

func TestFindImagesInHelmChartsOnNginxChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "ingress-nginx",
				Repo: repo.Entry{
					Name: "ingress-nginx",
					URL:  "https://kubernetes.github.io/ingress-nginx",
				},
				Version:        "4.8.3",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 2)
}

func TestFindImagesInHelmChartsOnReflectorChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "reflector",
				Repo: repo.Entry{
					Name: "reflector",
					URL:  "https://emberstack.github.io/helm-charts",
				},
				Version:        "7.1.216",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 1)
}

func TestFindImagesInHelmChartsOnVeleroChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "velero",
				Repo: repo.Entry{
					Name: "vmware-tanzu",
					URL:  "https://vmware-tanzu.github.io/helm-charts",
				},
				Version:        "5.1.4",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 1)
}

func TestFindImagesInHelmChartsOnKuredChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "kured",
				Repo: repo.Entry{
					Name: "kubereboot",
					URL:  "https://kubereboot.github.io/charts",
				},
				Version:        "5.3.1",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 1)
}

func TestFindImagesInHelmChartsOnKedaChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "keda",
				Repo: repo.Entry{
					Name: "kedacore",
					URL:  "https://kedacore.github.io/charts",
				},
				Version:        "2.12.1",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 2)
}

func TestFindImagesInHelmChartsOnTrivyOperatorChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "trivy-operator",
				Repo: repo.Entry{
					Name: "aquasecurity",
					URL:  "https://aquasecurity.github.io/helm-charts",
				},
				Version:        "0.19.0",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 2)
}

func TestFindImagesInHelmChartsOnKubescapeChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "kubescape-operator",
				Repo: repo.Entry{
					Name: "kubescape",
					URL:  "https://kubescape.github.io/helm-charts",
				},
				Version:        "1.16.3",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 13)
}

func TestFindImagesInHelmChartsOnKyvernoChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "kyverno",
				Repo: repo.Entry{
					Name: "kyverno",
					URL:  "https://kyverno.github.io/kyverno",
				},
				Version:        "3.1.1",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
			{
				Name: "kyverno-policies",
				Repo: repo.Entry{
					Name: "kyverno",
					URL:  "https://kyverno.github.io/kyverno",
				},
				Version:        "3.1.1",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 2, 5)
}

func TestFindImagesInHelmChartsOnArgoCDChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "argo-cd",
				Repo: repo.Entry{
					Name: "argoproj",
					URL:  "https://argoproj.github.io/argo-helm",
				},
				Version:        "5.51.4",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 2)
}

func TestFindImagesInHelmChartsOnHarborChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "harbor",
				Repo: repo.Entry{
					Name: "harbor",
					URL:  "https://helm.goharbor.io",
				},
				Version:        "1.14.1",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 8)
}

func TestFindImagesInHelmChartsOnExternalSecretsChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "external-secrets",
				Repo: repo.Entry{
					Name: "external-secrets",
					URL:  "https://charts.external-secrets.io",
				},
				Version:        "0.10.4",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 1, 2)
}

func TestFindImagesInHelmChartsOnKubePrometheusStackChart(t *testing.T) {
	t.Parallel()

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Arrange
	settings, err := testSettings()
	if err != nil {
		t.Error(err)
	}

	rc, _ := helm.NewDefaultRegistryClient()
	charts := helm.ChartCollection{
		Charts: []*helm.Chart{
			{
				Name: "kube-prometheus-stack",
				Repo: repo.Entry{
					Name: "prometheus-community",
					URL:  "https://prometheus-community.github.io/helm-charts",
				},
				Version:        "63.1.0",
				RegistryClient: rc,
				IndexFileLoader: &helm.FunctionLoader{
					LoadFunc: repo.LoadIndexFile,
				},
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
		IdentifyImages:  true,
		Settings:        settings,
	}
	_, err = co.ChartCollection.SetupHelm(settings)
	if err != nil {
		t.Error(err)
	}

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	assertChartData(t, data, 4, 10)
}
