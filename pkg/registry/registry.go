package registry

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	v1_spec "github.com/google/go-containerregistry/pkg/v1"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type Registry struct {
	Name         string
	URL          string
	Insecure     bool
	PlainHTTP    bool
	PrefixSource bool
}

type Exister interface {
	Exist(context.Context, string, string) (bool, error)
	GetName() string
}

var _ Exister = (*Registry)(nil)

type Puller interface {
	Pull(context.Context, string, string) (*v1.Descriptor, error)
}

var _ Puller = (*Registry)(nil)

type Pusher interface {
	Exister
	Push(ctx context.Context, sourceURL string, img string, tag string, arch *string) (v1.Descriptor, error)
}

var _ Pusher = (*Registry)(nil)

func (r Registry) GetName() string {
	return r.Name
}

func newDockerCredentialsStore() (*credentials.DynamicStore, error) {
	storeOpts := credentials.StoreOptions{}
	return credentials.NewStoreFromDocker(storeOpts)
}

func setupRepository(baseURL string, name string, credStore *credentials.DynamicStore) (*remote.Repository, error) {
	ref := strings.Join([]string{baseURL, name}, "/")
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, err
	}

	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore),
	}
	return repo, nil
}

func isLocalReference(url string) bool {
	return strings.Contains(url, "localhost") || strings.Contains(url, "0.0.0.0")
}

func parsePlatform(arch string) (*v1.Platform, error) {
	v, err := v1_spec.ParsePlatform(arch)
	if err != nil {
		return nil, err
	}
	return &v1.Platform{
		Architecture: v.Architecture,
		OS:           v.OS,
		OSVersion:    v.OSVersion,
		OSFeatures:   v.OSFeatures,
		Variant:      v.Variant,
	}, nil
}

// Push pushes an image to the registry.
func (r Registry) Push(ctx context.Context, sourceURL string, name string, tag string, arch *string) (v1.Descriptor, error) {
	credStore, err := newDockerCredentialsStore()
	if err != nil {
		return v1.Descriptor{}, err
	}

	source, err := setupRepository(sourceURL, name, credStore)
	if err != nil {
		return v1.Descriptor{}, err
	}

	source.PlainHTTP = isLocalReference(sourceURL)

	url, _ := strings.CutPrefix(r.URL, "oci://")
	if r.PrefixSource {
		noPorts := strings.Split(sourceURL, ":")[0]
		noTLD := strings.Split(noPorts, ".")[0]
		old := name
		name = fmt.Sprintf("%s/%s", noTLD, name)
		slog.Info("registry has PrefixSource enabled", slog.String("old", old), slog.String("new", name))
	}
	target, err := setupRepository(url, name, credStore)
	if err != nil {
		return v1.Descriptor{}, err
	}

	target.PlainHTTP = r.PlainHTTP

	opts := oras.DefaultCopyOptions
	if arch != nil {
		platform, err := parsePlatform(*arch)
		if err != nil {
			return v1.Descriptor{}, err
		}
		opts.WithTargetPlatform(platform)
	}

	manifest, err := oras.Copy(ctx, source, tag, target, tag, opts)
	if err != nil {
		return v1.Descriptor{}, err
	}

	return manifest, nil
}

func (r Registry) Fetch(ctx context.Context, name string, tag string) (*v1.Descriptor, error) {
	// 1. Connect to a remote repository
	url, _ := strings.CutPrefix(r.URL, "oci://")
	ref := strings.Join([]string{url, name}, "/")
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, err
	}

	repo.PlainHTTP = r.PlainHTTP

	// prepare authentication using Docker credentials
	storeOpts := credentials.StoreOptions{}
	credStore, err := credentials.NewStoreFromDocker(storeOpts)
	if err != nil {
		return nil, err
	}
	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore), // Use the credentials store
	}

	// 2. Copy from the remote repository to the OCI layout store
	d, err := repo.Resolve(ctx, tag)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func (r Registry) Pull(ctx context.Context, name string, tag string) (*v1.Descriptor, error) {
	// 0. Create an OCI layout store
	store := memory.New()

	// 1. Connect to a remote repository
	ref := strings.Join([]string{r.URL, name}, "/")
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, err
	}

	repo.PlainHTTP = r.PlainHTTP

	// prepare authentication using Docker credentials
	storeOpts := credentials.StoreOptions{}
	credStore, err := credentials.NewStoreFromDocker(storeOpts)
	if err != nil {
		return nil, err
	}
	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore), // Use the credentials store
	}

	// 2. Copy from the remote repository to the OCI layout store
	d, err := oras.Copy(ctx, repo, tag, store, tag, oras.DefaultCopyOptions)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func (r Registry) Exist(ctx context.Context, name string, tag string) (bool, error) {
	return Exist(ctx, strings.Join([]string{r.URL, name}, "/"), tag, r.PlainHTTP)
}

func Exists(ctx context.Context, ref string, tag string, registries []*Registry) map[string]bool {
	ref, _ = strings.CutPrefix(ref, "oci://")

	m := make(map[string]bool, len(registries))

	for _, r := range registries {
		exists := func(r Exister) bool {
			exists, err := r.Exist(ctx, ref, tag)
			if err != nil {
				return false
			}
			return exists
		}(r)

		m[r.URL] = exists
	}

	return m
}

func Exist(ctx context.Context, reference string, tag string, plainHTTP bool) (bool, error) {
	reference, _ = strings.CutPrefix(reference, "oci://")

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
