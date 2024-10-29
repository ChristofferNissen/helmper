package helm

import (
	"go.uber.org/fx"
	"helm.sh/helm/v3/pkg/repo"
)

type IndexFileLoader interface {
	LoadIndexFile(indexFilePath string) (*repo.IndexFile, error)
}

type DefaultIndexFileLoader struct{}

func (d *DefaultIndexFileLoader) LoadIndexFile(indexFilePath string) (*repo.IndexFile, error) {
	return repo.LoadIndexFile(indexFilePath)
}

var IndexFileLoaderModule = fx.Provide(FunctionLoader{LoadFunc: repo.LoadIndexFile})
