package helm

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/blang/semver/v4"
	"golang.org/x/xerrors"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func VersionsInRange(r semver.Range, c Chart) ([]string, error) {
	prefixV := strings.Contains(c.Version, "v")
	config := cli.New()
	indexPath := fmt.Sprintf("%s/%s-index.yaml", config.RepositoryCache, c.Repo.Name)
	index, err := c.IndexFileLoader.LoadIndexFile(indexPath)
	if err != nil {
		return nil, err
	}
	index.SortEntries()
	versions := index.Entries[c.Name]
	versionsInRange := []string{}
	for _, v := range versions {
		sv, err := semver.ParseTolerant(v.Version)
		if err != nil {
			continue
		}
		if len(sv.Pre) > 0 {
			continue
		}
		if r(sv) {
			s := sv.String()
			if prefixV {
				s = "v" + s
			}
			versionsInRange = append(versionsInRange, s)
		}
	}
	return versionsInRange, nil
}

func (c Chart) ResolveVersions() ([]string, error) {
	// prefixV := strings.Contains(c.Version, "v")
	version := strings.ReplaceAll(c.Version, "v", "")
	r, err := semver.ParseRange(version)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		ref := strings.TrimPrefix(strings.TrimSuffix(c.Repo.URL, "/")+"/"+c.Name, "oci://")
		versionsInRange, err := c.RegistryClient.Tags(ref)
		if err != nil {
			return nil, err
		}
		return versionsInRange, nil
	}

	update, err := c.addToHelmRepositoryFile()
	if err != nil {
		return nil, err
	}
	if update {
		_, err = updateRepositories(false, false)
		if err != nil {
			return nil, err
		}
	}
	return VersionsInRange(r, c)
}

func (c Chart) ResolveVersion() (string, error) {
	v := strings.ReplaceAll(c.Version, "*", "x")
	r, err := semver.ParseRange(v)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		ref := strings.TrimPrefix(strings.TrimSuffix(c.Repo.URL, "/")+"/"+c.Name, "oci://")
		vs, err := c.RegistryClient.Tags(ref)
		if err != nil {
			return "", err
		}

		if len(vs) > 0 {
			return vs[len(vs)-1], nil
		}
		return "", xerrors.Errorf("Not found")
	}

	update, err := c.addToHelmRepositoryFile()
	if err != nil {
		return "", err
	}
	if update {
		_, err = updateRepositories(false, false)
		if err != nil {
			return "", err
		}
	}

	config := cli.New()
	indexPath := fmt.Sprintf("%s/%s-index.yaml", config.RepositoryCache, c.Repo.Name)
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return "", err
	}
	index.SortEntries()
	versions := index.Entries[c.Name]
	for _, v := range versions {
		sv, err := semver.ParseTolerant(v.Version)
		if err != nil {
			continue
		}
		if len(sv.Pre) > 0 {
			continue
		}
		if r(sv) {
			slog.Debug("Resolved chart version", slog.String("chart", c.Name), slog.String("version", sv.String()))
			return sv.String(), nil
		}
	}
	return "", xerrors.New("Not Found")
}

func (c Chart) LatestVersion() (string, error) {
	config := cli.New()

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		ref := strings.TrimPrefix(strings.TrimSuffix(c.Repo.URL, "/")+"/"+c.Name, "oci://")
		repo, err := remote.NewRepository(ref)
		if err != nil {
			return "", err
		}
		repo.PlainHTTP = c.PlainHTTP

		storeOpts := credentials.StoreOptions{}
		credStore, err := credentials.NewStoreFromDocker(storeOpts)
		if err != nil {
			return "", err
		}
		repo.Client = &auth.Client{
			Client:     retry.DefaultClient,
			Cache:      auth.NewCache(),
			Credential: credentials.Credential(credStore),
		}

		vPrefix := strings.Contains(c.Version, "v")
		vs, err := c.RegistryClient.Tags(ref)

		if err != nil {
			return "", err
		}
		l := vs[len(vs)-1]
		if vPrefix {
			l = "v" + l
		}
		return l, nil
	}

	indexPath := fmt.Sprintf("%s/%s-index.yaml", config.RepositoryCache, c.Repo.Name)
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return "", err
	}
	index.SortEntries()
	res := "Not Found"
	versions := index.Entries[c.Name]
	for _, v := range versions {
		sv, err := semver.Parse(v.Version)
		if err != nil {
			res = v.Version
			break
		}
		if len(sv.Pre) == 0 {
			res = sv.String()
			break
		}
	}
	return res, nil
}

type FunctionLoader struct {
	LoadFunc func(indexFilePath string) (*repo.IndexFile, error)
}

func (fl *FunctionLoader) LoadIndexFile(indexFilePath string) (*repo.IndexFile, error) {
	return fl.LoadFunc(indexFilePath)
}
