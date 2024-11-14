package helm

import (
	"context"
	"strings"

	helm_registry "helm.sh/helm/v3/pkg/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type OCIRegistryClient struct {
	client    RegistryClient
	PlainHTTP bool
}

func NewOCIRegistryClient(client RegistryClient, plainHTTP bool) *OCIRegistryClient {
	return &OCIRegistryClient{
		client:    client,
		PlainHTTP: plainHTTP,
	}
}

func (c *OCIRegistryClient) Pull(ref string, opts ...helm_registry.PullOption) (*helm_registry.PullResult, error) {
	return c.client.Pull(ref, opts...)
}

func (c *OCIRegistryClient) Push(chart []byte, destination string, opts ...helm_registry.PushOption) (*helm_registry.PushResult, error) {
	return c.client.Push(chart, destination, opts...)
}

func (c *OCIRegistryClient) Tags(ref string) ([]string, error) {
	ref = strings.TrimPrefix(strings.TrimSuffix(ref, "/"), "oci://")
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return []string{}, err
	}
	repo.PlainHTTP = c.PlainHTTP

	storeOpts := credentials.StoreOptions{}
	credStore, err := credentials.NewStoreFromDocker(storeOpts)
	if err != nil {
		return []string{}, err
	}
	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore), // Use the credentials store
	}

	vs := []string{}
	err = repo.Tags(context.TODO(), "", func(tags []string) error {
		vs = append(vs, tags...)
		return nil
	})
	if err != nil {
		return []string{}, err
	}

	return vs, nil
}
