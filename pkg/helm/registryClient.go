package helm

import (
	"go.uber.org/fx"
	helm_registry "helm.sh/helm/v3/pkg/registry"
)

// Define the interface for the registry client
type RegistryClient interface {
	Pull(ref string, opts ...helm_registry.PullOption) (*helm_registry.PullResult, error)
	Push(chart []byte, destination string, opts ...helm_registry.PushOption) (*helm_registry.PushResult, error)
	Tags(ref string) ([]string, error)
}

// Default registry client provider
func NewDefaultRegistryClient() (RegistryClient, error) {
	plainHTTP := false
	debug := false

	return NewRegistryClient(plainHTTP, debug)
}

func NewRegistryClient(plainHTTP, debug bool) (RegistryClient, error) {
	opts := []helm_registry.ClientOption{}
	if plainHTTP {
		opts = append(opts, helm_registry.ClientOptPlainHTTP())
	}
	if debug {
		opts = append(opts, helm_registry.ClientOptDebug(true))
	}
	return helm_registry.NewClient(opts...)
}

var RegistryModule = fx.Provide(NewDefaultRegistryClient)
