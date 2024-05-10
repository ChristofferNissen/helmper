package registry

import (
	"context"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type Registry struct {
	Name      string
	URL       string
	Insecure  bool
	PlainHTTP bool
}

type Exister interface {
	Exist(context.Context, string, string) (bool, error)
	GetName() string
}

var _ Exister = (*Registry)(nil)

type Puller interface {
	Pull(context.Context, string, string) (bool, error)
}

var _ Puller = (*Registry)(nil)

type Pusher interface {
	Exister
	Push(ctx context.Context, sourceURL string, img string, tag string) (v1.Descriptor, error)
}

var _ Pusher = (*Registry)(nil)

func (r Registry) GetName() string {
	return r.Name
}

func (r Registry) Push(ctx context.Context, sourceURL string, name string, tag string) (v1.Descriptor, error) {

	// 1. Connect to a remote repository
	ref := strings.Join([]string{sourceURL, name}, "/")
	source, err := remote.NewRepository(ref)
	if err != nil {
		return v1.Descriptor{}, err
	}

	// Determine HTTP or HTTPS. Allow HTTP if local reference
	source.PlainHTTP = strings.Contains(sourceURL, "localhost") || strings.Contains(sourceURL, "0.0.0.0")
	// 3. Connect to our target repository
	image := strings.Join([]string{r.URL, name}, "/")

	target, err := remote.NewRepository(image)
	if err != nil {
		return v1.Descriptor{}, err
	}
	// prepare authentication using Docker credentials
	storeOpts := credentials.StoreOptions{}
	credStore, err := credentials.NewStoreFromDocker(storeOpts)
	if err != nil {
		return v1.Descriptor{}, err
	}
	target.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore), // Use the credentials store
	}

	// todo: check if user specified auth
	target.PlainHTTP = r.PlainHTTP
	manifest, err := oras.Copy(ctx, source, tag, target, tag, oras.DefaultCopyOptions)
	if err != nil {
		return v1.Descriptor{}, err
	}

	return manifest, nil
}

func (r Registry) Pull(ctx context.Context, name string, tag string) (bool, error) {
	// 0. Create an OCI layout store
	store := memory.New()

	// 1. Connect to a remote repository
	ref := strings.Join([]string{r.URL, name}, "/")
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return false, err
	}

	repo.PlainHTTP = r.PlainHTTP

	// prepare authentication using Docker credentials
	storeOpts := credentials.StoreOptions{}
	credStore, err := credentials.NewStoreFromDocker(storeOpts)
	if err != nil {
		return false, err
	}
	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore), // Use the credentials store
	}

	// 2. Copy from the remote repository to the OCI layout store
	_, err = oras.Copy(ctx, repo, tag, store, tag, oras.DefaultCopyOptions)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r Registry) Exist(ctx context.Context, name string, tag string) (bool, error) {
	return Exist(ctx, strings.Join([]string{r.URL, name}, "/"), tag, r.PlainHTTP)
}

func Exists(ctx context.Context, img *Image, registries []Registry) (map[string]bool, error) {
	m := make(map[string]bool, len(registries))

	for _, r := range registries {
		exists, err := func(img *Image, r Exister) (bool, error) {
			name, err := img.ImageName()
			if err != nil {
				return false, err
			}
			exists, err := r.Exist(ctx, name, img.Tag)
			if err != nil {
				return false, err
			}
			return exists, nil
		}(img, r)
		if err != nil {
			return nil, err
		}

		m[r.GetName()] = exists
	}

	return m, nil
}

func Exist(ctx context.Context, reference string, tag string, plainHTTP bool) (bool, error) {

	// 1. Connect to a remote repository
	repo, err := remote.NewRepository(reference)
	if err != nil {
		return false, err
	}

	repo.PlainHTTP = plainHTTP

	// prepare authentication using Docker credentials
	storeOpts := credentials.StoreOptions{}
	credStore, err := credentials.NewStoreFromDocker(storeOpts)
	if err != nil {
		return false, err
	}
	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore), // Use the credentials store
	}

	// 2. Copy from the remote repository to the OCI layout store
	opts := oras.DefaultFetchOptions
	_, _, err = oras.Fetch(ctx, repo, tag, opts)
	return err == nil, err
}
