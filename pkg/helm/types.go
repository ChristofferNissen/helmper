package helm

import (
	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

type Images struct {
	Exclude []struct {
		Ref string `json:"ref"`
	} `json:"exclude"`
	ExcludeCopacetic []struct {
		Ref string `json:"ref"`
	} `json:"excludeCopacetic"`
	Modify []struct {
		From          string `json:"from"`
		FromValuePath string `json:"fromValuePath"`
		To            string `json:"to"`
	} `json:"modify"`
}

type Chart struct {
	Name            string         `json:"name"`
	Version         string         `json:"version"`
	ValuesFilePath  string         `json:"valuesFilePath"`
	Values          map[string]any `json:"values,omitempty"`
	Repo            repo.Entry     `json:"repo"`
	Parent          *Chart
	Images          *Images `json:"images"`
	PlainHTTP       bool    `json:"plainHTTP"`
	DepsCount       int
	RegistryClient  RegistryClient
	IndexFileLoader IndexFileLoader
}

type ChartCollection struct {
	Charts []*Chart `json:"charts"`
}

// channels to share data between goroutines
type chartInfo struct {
	chartRef *chart.Chart
	*Chart
}

type imageInfo struct {
	available  bool
	chart      *Chart
	image      *image.Image
	collection *[]string
}

type ChartData map[*Chart]map[*image.Image][]string

type RegistryChartStatus map[*registry.Registry]map[*Chart]bool

type RegistryImageStatus map[*registry.Registry]map[*image.Image]bool

type Mirror struct {
	Registry string `yaml:"registry"`
	Mirror   string `yaml:"mirror"`
}
