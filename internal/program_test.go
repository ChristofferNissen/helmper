package internal

import (
	"context"
	"testing"

	"github.com/ChristofferNissen/helmper/helmper/pkg/helm"
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

func TestFindImagesInHelmChartsOnPrometheusChart(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := context.TODO()
	// home, _ := os.UserHomeDir()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "prometheus",
				RepoName: "prometheus-community",
				URL:      "https://prometheus-community.github.io/helm-charts",
				Version:  "25.8.0",
				// ValuesFilePath: filepath.Join(home, "repos/helmper/helmper/cmd/helmper/values/prometheus/values.yaml"),
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "promtail",
				RepoName: "grafana",
				URL:      "https://grafana.github.io/helm-charts",
				Version:  "6.15.3",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "loki",
				RepoName: "grafana",
				URL:      "https://grafana.github.io/helm-charts",
				Version:  "5.38.0",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "mimir-distributed",
				RepoName: "grafana",
				URL:      "https://grafana.github.io/helm-charts",
				Version:  "5.1.3",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "grafana",
				RepoName: "grafana",
				URL:      "https://grafana.github.io/helm-charts",
				Version:  "7.0.9",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "cilium",
				RepoName: "cilium",
				URL:      "https://helm.cilium.io/",
				Version:  "1.14.4",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "cert-manager",
				RepoName: "cert-manager",
				URL:      "https://charts.jetstack.io",
				Version:  "v1.13.2",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "ingress-nginx",
				RepoName: "ingress-nginx",
				URL:      "https://kubernetes.github.io/ingress-nginx",
				Version:  "4.8.3",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "reflector",
				RepoName: "reflector",
				URL:      "https://emberstack.github.io/helm-charts",
				Version:  "7.1.216",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "velero",
				RepoName: "vmware-tanzu",
				URL:      "https://vmware-tanzu.github.io/helm-charts",
				Version:  "5.1.4",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "kured",
				RepoName: "kubereboot",
				URL:      "https://kubereboot.github.io/charts",
				Version:  "5.3.1",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "keda",
				RepoName: "kedacore",
				URL:      "https://kedacore.github.io/charts",
				Version:  "2.12.1",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

// Security

func TestFindImagesInHelmChartsOnTrivyOperatorChart(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "trivy-operator",
				RepoName: "aquasecurity",
				URL:      "https://aquasecurity.github.io/helm-charts",
				Version:  "0.19.0",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "kubescape-operator",
				RepoName: "kubescape",
				URL:      "https://kubescape.github.io/helm-charts",
				Version:  "1.16.3",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "kyverno",
				RepoName: "kyverno",
				URL:      "https://kyverno.github.io/kyverno",
				Version:  "3.1.1",
			},
			{
				Name:     "kyverno-policies",
				RepoName: "kyverno",
				URL:      "https://kyverno.github.io/kyverno",
				Version:  "3.1.1",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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

// CI/CD

func TestFindImagesInHelmChartsOnArgoCDChart(t *testing.T) {
	t.Parallel()

	// Arrange
	ctx := context.TODO()

	charts := helm.ChartCollection{
		Charts: []helm.Chart{
			{
				Name:     "argo-cd",
				RepoName: "argoproj",
				URL:      "https://argoproj.github.io/argo-helm",
				Version:  "5.51.4",
			},
		},
	}

	co := helm.ChartOption{
		ChartCollection: &charts,
	}
	err := co.ChartCollection.SetupHelm()
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
