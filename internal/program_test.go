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

// Charts tested for number of charts (charts and subcharts) and number of images below:
// Prometheus
// Promtail
// Loki
// Mimir-Distributed
// Grafana
// Cilium
// Cert-Manager
// Ingress-Nginx
// Reflector
// Velero
// Kured
// Keda
// Trivy-Operator
// Kubescape-Operator
// ArgoCD
// Harbor

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

	expectedChartCount := 0
	expectedImageCount := 0

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 5
	expectedImageCount := 6

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 1

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 2
	expectedImageCount := 6

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 3
	expectedImageCount := 9

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 5

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 5

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 5

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 2

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
}

// Cluster

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

	expectedChartCount := 1
	expectedImageCount := 1

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 2

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 1

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 3

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 3

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 17

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 10

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 2

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 10

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 1
	expectedImageCount := 3

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
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

	expectedChartCount := 4
	expectedImageCount := 13

	// Act
	data, err := co.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if len(data) != expectedChartCount {
		t.Fatalf("want '%d' number of charts, got '%d'\n", expectedChartCount, len(data))
	}

	imageCount := 0
	for _, images := range data {
		imageCount = imageCount + len(images)
	}

	if imageCount != expectedImageCount {
		t.Fatalf("want '%d' number of images, got '%d'\n", expectedImageCount, imageCount)
	}
}
