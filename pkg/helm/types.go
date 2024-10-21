package helm

import (
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"go.uber.org/fx"
	"helm.sh/helm/v3/pkg/chart"
	helm_registry "helm.sh/helm/v3/pkg/registry"
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

type IndexFileLoader interface {
	LoadIndexFile(indexFilePath string) (*repo.IndexFile, error)
}

type DefaultIndexFileLoader struct{}

func (d *DefaultIndexFileLoader) LoadIndexFile(indexFilePath string) (*repo.IndexFile, error) {
	return repo.LoadIndexFile(indexFilePath)
}

var IndexFileLoaderModule = fx.Provide(FunctionLoader{LoadFunc: repo.LoadIndexFile})

// Define the interface for the registry client
type RegistryClient interface {
	Push(chart []byte, destination string, opts ...helm_registry.PushOption) (*helm_registry.PushResult, error)
	Tags(ref string) ([]string, error)
}

// Default registry client provider
func NewDefaultRegistryClient() (RegistryClient, error) {
	return helm_registry.NewClient(
		helm_registry.ClientOptDebug(true),
		helm_registry.ClientOptPlainHTTP(),
	)
}

var RegistryModule = fx.Provide(NewDefaultRegistryClient)

type Chart struct {
	Name            string     `json:"name"`
	Version         string     `json:"version"`
	ValuesFilePath  string     `json:"valuesFilePath"`
	Repo            repo.Entry `json:"repo"`
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
	image      *registry.Image
	collection *[]string
}

type ChartData map[Chart]map[*registry.Image][]string

type RegistryChartStatus map[*registry.Registry]map[*Chart]bool

type RegistryImageStatus map[*registry.Registry]map[*registry.Image]bool

type Mirror struct {
	Registry string `yaml:"registry"`
	Mirror   string `yaml:"mirror"`
}
