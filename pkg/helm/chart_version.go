package helm

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/blang/semver/v4"
	"golang.org/x/xerrors"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
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

func (c Chart) ResolveVersions(settings *cli.EnvSettings) ([]string, error) {
	version := strings.ReplaceAll(c.Version, "v", "")
	r, err := semver.ParseRange(version)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		url, _ := strings.CutPrefix(c.Repo.URL, "oci://")
		tags, err := c.RegistryClient.Tags(url)
		if err != nil {
			return nil, err
		}

		vs := []semver.Version{}
		for _, t := range tags {
			s, err := semver.ParseTolerant(t)
			if err != nil {
				// non semver tag
				continue
			}
			vs = append(vs, s)
		}

		semver.Sort(vs)

		prefixV := strings.Contains(c.Version, "v")
		versionsInRange := []string{}
		for _, v := range vs {
			if len(v.Pre) > 0 {
				continue
			}
			if r(v) {
				s := v.String()
				if prefixV {
					s = "v" + s
				}
				versionsInRange = append(versionsInRange, s)
			}
		}
		return versionsInRange, nil
	}

	update, err := c.addToHelmRepositoryFile(settings)
	if err != nil {
		return nil, err
	}
	if update {
		_, err = updateRepositories(settings, false, false)
		if err != nil {
			return nil, err
		}
	}
	return VersionsInRange(r, c)
}

func (c Chart) ResolveVersion(settings *cli.EnvSettings) (string, error) {
	v := strings.ReplaceAll(c.Version, "*", "x")
	r, err := semver.ParseRange(v)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		url, _ := strings.CutPrefix(c.Repo.URL, "oci://")
		tags, err := c.RegistryClient.Tags(url)
		if err != nil {
			return "", err
		}

		vs := []semver.Version{}
		for _, t := range tags {
			s, err := semver.ParseTolerant(t)
			if err != nil {
				// non semver tag
				continue
			}
			vs = append(vs, s)
		}

		semver.Sort(vs)

		vs2 := []string{}
		for _, v := range vs {
			if len(v.Pre) > 0 {
				continue
			}
			if r(v) {
				vs2 = append(vs2, v.String())
			}
		}

		prefixV := strings.Contains(c.Version, "v")
		if prefixV {
			return "v" + vs2[len(vs2)-1], nil
		}
		return vs2[len(vs2)-1], nil
	}

	update, err := c.addToHelmRepositoryFile(settings)
	if err != nil {
		return "", err
	}
	if update {
		_, err = updateRepositories(settings, false, false)
		if err != nil {
			return "", err
		}
	}

	indexPath := fmt.Sprintf("%s/%s-index.yaml", settings.RepositoryCache, c.Repo.Name)
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

func (c Chart) LatestVersion(settings *cli.EnvSettings) (string, error) {

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		url, _ := strings.CutPrefix(c.Repo.URL, "oci://")
		vPrefix := strings.Contains(c.Version, "v")
		tags, err := c.RegistryClient.Tags(url)
		if err != nil {
			return "", err
		}

		vs := []semver.Version{}
		for _, t := range tags {
			s, err := semver.ParseTolerant(t)
			if err != nil {
				// non semver tag
				continue
			}
			vs = append(vs, s)
		}

		semver.Sort(vs)

		if vPrefix {
			return "v" + vs[len(vs)-1].String(), nil
		}
		return vs[len(vs)-1].String(), nil
	}

	indexPath := fmt.Sprintf("%s/%s-index.yaml", settings.RepositoryCache, c.Repo.Name)
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
