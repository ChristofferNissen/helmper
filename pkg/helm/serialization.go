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

type Output struct {
	Charts  []Chart               `yaml:"charts"`
	Images  []string              `yaml:"images"`
	Mapping SerializableChartData `yaml:"mapping"`
}

type SerializableChartData map[string][]string
type SerializableCharts map[string]Chart

func toSerializable(data ChartData) Output {
	// serializableCharts := make(SerializableCharts)
	charts := make([]Chart, 0)
	imgs := make([]string, 0)
	serializableMapping := make(SerializableChartData)

	seen := make([]*image.Image, 0)
	for chart, imageMap := range data {
		serializableImages := make([]string, 0)
		for img := range imageMap {
			s := img.String()
			if !img.InP(seen) {
				seen = append(seen, img)
				imgs = append(imgs, s)
			}
			serializableImages = append(serializableImages, s)
		}
		key := filepath.Join(chart.Repo.URL, chart.Name, chart.Version)
		// serializableCharts[key] = chart
		serializableMapping[key] = serializableImages
		charts = append(charts, chart)
	}
	return Output{Charts: charts, Images: imgs, Mapping: serializableMapping}
}

func fromSerializable(output Output, rc RegistryClient) ChartData {
	originalData := make(ChartData)

	data := output.Charts
	mapping := output.Mapping

	for key, serializableImages := range mapping {
		chart := func() Chart {
			for _, c := range data {
				s := filepath.Join(c.Repo.URL, c.Name, c.Version)
				if key == s {
					return c
				}
			}
			return Chart{}
		}()

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
	fp := filename + ".yaml"

	output := toSerializable(data)

	// Write charts
	chartsData, err := yaml.Marshal(output)
	if err != nil {
		return err
	}
	err = os.WriteFile(fp, chartsData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func ReadYAMLFromFile(filename string, rc RegistryClient) (ChartData, error) {
	fp := filename + ".yaml"

	// Read charts
	var output Output
	chartsData, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(chartsData, &output)
	if err != nil {
		return nil, err
	}

	return fromSerializable(output, rc), nil
}

type StatusOutput struct {
	ChartStatus RegistryChartStatusOutput     `yaml:"chartStatus"`
	ImageStatus RegistryImageStatusOutput     `yaml:"imageStatus"`
	Registries  map[string]*registry.Registry `yaml:"registries"`
}

type RegistryChartStatusOutput struct {
	Status map[string]map[string]bool `yaml:"status"`
	Charts map[string]*Chart          `yaml:"charts"`
}

type RegistryImageStatusOutput struct {
	Status map[string]map[string]bool `yaml:"status"`
	Images map[string]*image.Image    `yaml:"images"`
}

func WriteStatusOutputToYAML(filename string, chartStatus RegistryChartStatus, imageStatus RegistryImageStatus) error {

	serializableRegistries := make(map[string]*registry.Registry)

	// Split chart status data
	serializableChartStatus := make(map[string]map[string]bool)
	serializableCharts := make(map[string]*Chart)
	for registry, chartMap := range chartStatus {
		registryKey := registry.Name + ":" + registry.URL
		serializableRegistries[registryKey] = registry
		serializableChartMap := make(map[string]bool)
		for chart, status := range chartMap {
			chartKey := chart.Name + ":" + chart.Version
			serializableChartMap[chartKey] = status
			serializableCharts[chartKey] = chart
		}
		serializableChartStatus[registryKey] = serializableChartMap
	}

	// Split image status data
	serializableImageStatus := make(map[string]map[string]bool)
	serializableImages := make(map[string]*image.Image)
	for registry, imageMap := range imageStatus {
		registryKey := registry.Name + ":" + registry.URL
		serializableRegistries[registryKey] = registry
		serializableImageMap := make(map[string]bool)
		for img, status := range imageMap {
			imageKey := img.Registry + "/" + img.Repository + ":" + img.Tag
			serializableImageMap[imageKey] = status
			serializableImages[imageKey] = img
		}
		serializableImageStatus[registryKey] = serializableImageMap
	}

	output := StatusOutput{
		ChartStatus: RegistryChartStatusOutput{
			Status: serializableChartStatus,
			Charts: serializableCharts,
		},
		ImageStatus: RegistryImageStatusOutput{
			Status: serializableImageStatus,
			Images: serializableImages,
		},
		Registries: serializableRegistries,
	}

	// Write status
	statusData, err := yaml.Marshal(output)
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, statusData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func ReadStatusOutputFromYAML(filename string) (RegistryChartStatus, RegistryImageStatus, error) {

	// Read status
	var output StatusOutput
	statusData, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}
	err = yaml.Unmarshal(statusData, &output)
	if err != nil {
		return nil, nil, err
	}

	// Convert chart status back to original type
	chartStatus := make(RegistryChartStatus)
	for registryKey, serializableChartMap := range output.ChartStatus.Status {
		registry := output.Registries[registryKey]
		chartMap := make(map[*Chart]bool)
		for chartKey, status := range serializableChartMap {
			chart := output.ChartStatus.Charts[chartKey]
			chartMap[chart] = status
		}
		chartStatus[registry] = chartMap
	}

	// Convert image status back to original type
	imageStatus := make(RegistryImageStatus)
	for registryKey, serializableImageMap := range output.ImageStatus.Status {
		registry := output.Registries[registryKey]
		imageMap := make(map[*image.Image]bool)
		for imageKey, status := range serializableImageMap {
			img := output.ImageStatus.Images[imageKey]
			imageMap[img] = status
		}
		imageStatus[registry] = imageMap
	}

	return chartStatus, imageStatus, nil
}
