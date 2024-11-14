package helm

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/repo"
)

type SerializableChartData map[string][]string
type SerializableCharts map[string]Chart

func toSerializable(data ChartData) (SerializableCharts, SerializableChartData) {
	serializableCharts := make(SerializableCharts)
	serializableData := make(SerializableChartData)

	for chart, imageMap := range data {
		serializableImages := make([]string, 0)
		for img := range imageMap {
			serializableImages = append(serializableImages, img.String())
		}
		key := filepath.Join(chart.Repo.URL, chart.Name, chart.Version)
		serializableCharts[key] = chart
		serializableData[key] = serializableImages
	}
	return serializableCharts, serializableData
}

func fromSerializable(charts SerializableCharts, data SerializableChartData, rc RegistryClient) ChartData {
	originalData := make(ChartData)
	for chart, serializableImages := range data {
		chart := charts[chart]
		imageMap := make(map[*image.Image][]string)
		for _, imgStr := range serializableImages {
			img, _ := image.RefToImage(imgStr)
			imageMap[to.Ptr(img)] = make([]string, 0)
		}
		if strings.HasPrefix(chart.Repo.URL, "oci://") {
			chart.RegistryClient = NewOCIRegistryClient(rc, chart.PlainHTTP)
		} else {
			chart.RegistryClient = rc
		}
		chart.IndexFileLoader = &FunctionLoader{
			LoadFunc: repo.LoadIndexFile,
		}
		originalData[chart] = imageMap
	}
	return originalData
}

func WriteYAMLToFile(filename string, data ChartData) error {
	chartsFile := filename + "-charts.yaml"
	imagesFile := filename + "-images.yaml"

	serializableCharts, serializableData := toSerializable(data)

	// Write charts
	chartsData, err := yaml.Marshal(serializableCharts)
	if err != nil {
		return err
	}
	err = os.WriteFile(chartsFile, chartsData, 0644)
	if err != nil {
		return err
	}

	// Write images
	imagesData, err := yaml.Marshal(serializableData)
	if err != nil {
		return err
	}
	err = os.WriteFile(imagesFile, imagesData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func ReadYAMLFromFile(filename string, rc RegistryClient) (ChartData, error) {
	chartsFile := filename + "-charts.yaml"
	imagesFile := filename + "-images.yaml"

	// Read charts
	var serializableCharts SerializableCharts
	chartsData, err := os.ReadFile(chartsFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(chartsData, &serializableCharts)
	if err != nil {
		return nil, err
	}

	// Read images
	var serializableData SerializableChartData
	imagesData, err := os.ReadFile(imagesFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(imagesData, &serializableData)
	if err != nil {
		return nil, err
	}

	return fromSerializable(serializableCharts, serializableData, rc), nil
}

func WriteRegistryChartStatusToYAML(filename string, data RegistryChartStatus) error {
	statusFile := filename + "-status.yaml"
	chartsFile := filename + "-charts-index.yaml"
	registryFile := filename + "-registry-index.yaml"

	// Split data
	serializableStatus := make(map[string]map[string]bool)
	serializableCharts := make(map[string]*Chart)
	serializableRegistries := make(map[string]*registry.Registry)
	for registry, chartMap := range data {
		registryKey := registry.Name + ":" + registry.URL
		serializableRegistries[registryKey] = registry
		serializableChartMap := make(map[string]bool)
		for chart, status := range chartMap {
			chartKey := chart.Name + ":" + chart.Version
			serializableChartMap[chartKey] = status
			serializableCharts[chartKey] = chart
		}
		serializableStatus[registryKey] = serializableChartMap
	}

	// Write status
	statusData, err := yaml.Marshal(serializableStatus)
	if err != nil {
		return err
	}
	err = os.WriteFile(statusFile, statusData, 0644)
	if err != nil {
		return err
	}

	// Write charts
	chartsData, err := yaml.Marshal(serializableCharts)
	if err != nil {
		return err
	}
	err = os.WriteFile(chartsFile, chartsData, 0644)
	if err != nil {
		return err
	}

	// Write registries
	registryData, err := yaml.Marshal(serializableRegistries)
	if err != nil {
		return err
	}
	err = os.WriteFile(registryFile, registryData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func WriteRegistryImageStatusToYAML(filename string, data RegistryImageStatus) error {
	statusFile := filename + "-status.yaml"
	imagesFile := filename + "-images-index.yaml"
	registryFile := filename + "-registry-index.yaml"

	// Split data
	serializableStatus := make(map[string]map[string]bool)
	serializableImages := make(map[string]*image.Image)
	serializableRegistries := make(map[string]*registry.Registry)
	for registry, imageMap := range data {
		registryKey := registry.Name + ":" + registry.URL
		serializableRegistries[registryKey] = registry
		serializableImageMap := make(map[string]bool)
		for img, status := range imageMap {
			imageKey := img.Registry + "/" + img.Repository + ":" + img.Tag
			serializableImageMap[imageKey] = status
			serializableImages[imageKey] = img
		}
		serializableStatus[registryKey] = serializableImageMap
	}

	// Write status
	statusData, err := yaml.Marshal(serializableStatus)
	if err != nil {
		return err
	}
	err = os.WriteFile(statusFile, statusData, 0644)
	if err != nil {
		return err
	}

	// Write images
	imagesData, err := yaml.Marshal(serializableImages)
	if err != nil {
		return err
	}
	err = os.WriteFile(imagesFile, imagesData, 0644)
	if err != nil {
		return err
	}

	// Write registries
	registryData, err := yaml.Marshal(serializableRegistries)
	if err != nil {
		return err
	}
	err = os.WriteFile(registryFile, registryData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func ReadRegistryChartStatusFromYAML(filename string) (RegistryChartStatus, error) {
	statusFile := filename + "-status.yaml"
	chartsFile := filename + "-charts-index.yaml"
	registryFile := filename + "-registry-index.yaml"

	// Read status
	var serializableStatus map[string]map[string]bool
	statusData, err := os.ReadFile(statusFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(statusData, &serializableStatus)
	if err != nil {
		return nil, err
	}

	// Read charts
	var serializableCharts map[string]*Chart
	chartsData, err := os.ReadFile(chartsFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(chartsData, &serializableCharts)
	if err != nil {
		return nil, err
	}

	// Read registries
	var serializableRegistries map[string]*registry.Registry
	registryData, err := os.ReadFile(registryFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(registryData, &serializableRegistries)
	if err != nil {
		return nil, err
	}

	// Convert back to original type
	data := make(RegistryChartStatus)
	for registryKey, serializableChartMap := range serializableStatus {
		registry := serializableRegistries[registryKey]
		chartMap := make(map[*Chart]bool)
		for chartKey, status := range serializableChartMap {
			chart := serializableCharts[chartKey]
			chartMap[chart] = status
		}
		data[registry] = chartMap
	}

	return data, nil
}

func ReadRegistryImageStatusFromYAML(filename string) (RegistryImageStatus, error) {
	statusFile := filename + "-status.yaml"
	imagesFile := filename + "-images-index.yaml"
	registryFile := filename + "-registry-index.yaml"

	// Read status
	var serializableStatus map[string]map[string]bool
	statusData, err := os.ReadFile(statusFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(statusData, &serializableStatus)
	if err != nil {
		return nil, err
	}

	// Read images
	var serializableImages map[string]*image.Image
	imagesData, err := os.ReadFile(imagesFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(imagesData, &serializableImages)
	if err != nil {
		return nil, err
	}

	// Read registries
	var serializableRegistries map[string]*registry.Registry
	registryData, err := os.ReadFile(registryFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(registryData, &serializableRegistries)
	if err != nil {
		return nil, err
	}

	// Convert back to original type
	data := make(RegistryImageStatus)
	for registryKey, serializableImageMap := range serializableStatus {
		registry := serializableRegistries[registryKey]
		imageMap := make(map[*image.Image]bool)
		for imageKey, status := range serializableImageMap {
			img := serializableImages[imageKey]
			imageMap[img] = status
		}
		data[registry] = imageMap
	}

	return data, nil
}
