package helm

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ChristofferNissen/helmper/pkg/util/file"
	"golang.org/x/xerrors"

	"github.com/coreos/go-semver/semver"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/repo"
)

type Chart struct {
	Name           string `json:"name"`
	RepoName       string `json:"repoName"`
	URL            string `json:"url"`
	Version        string `json:"version"`
	ValuesFilePath string `json:"valuesFilePath"`
	Parent         *Chart
}

// AddChartRepositoryToHelmRepositoryFile adds repository to Helm repository.yml to enable querying/pull
func (c Chart) AddToHelmRepositoryFile() error {
	config := cli.New()
	repoConfig := config.RepositoryConfig

	var f *repo.File = repo.NewFile()
	if file.Exists(repoConfig) {
		file, err := repo.LoadFile(repoConfig)
		if err != nil {
			return err
		}
		f = file
	}

	if !f.Has(c.RepoName) {
		f.Update(&repo.Entry{
			Name: c.RepoName,
			URL:  c.URL,
		})
		return f.WriteFile(repoConfig, 0644)
	}

	return nil
}

func (c Chart) ResolveVersion() (string, error) {

	if !strings.Contains(c.Version, "*") {
		return c.Version, nil
	}

	v := strings.Replace(c.Version, "*", "0", 1)
	s := semver.New(v)
	major := s.Major
	minor := s.Minor

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	indexPath := fmt.Sprintf("%s/.cache/helm/repository/%s-index.yaml", home, c.RepoName)
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return "", err
	}

	index.SortEntries()

	versions := index.Entries[c.Name]
	for _, v := range versions {

		sv, err := semver.NewVersion(v.Version)
		if err != nil {
			// not semver
			continue
		}

		switch {
		case sv.PreRelease != "":
			continue
		case sv.Major > major:
			continue
		case sv.Minor > minor:
			continue
		case sv.Major == major && sv.Minor == minor:
			slog.Debug("Resolved chart version", slog.String("chart", c.Name), slog.String("version", sv.String()))
			return sv.String(), nil
		}

	}

	return "", xerrors.New("Not Found")
}

func (c Chart) LatestVersion() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	indexPath := fmt.Sprintf("%s/.cache/helm/repository/%s-index.yaml", home, c.RepoName)
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return "", err
	}

	index.SortEntries()

	res := "Not Found"
	versions := index.Entries[c.Name]
	for _, v := range versions {

		sv, err := semver.NewVersion(v.Version)
		if err != nil {
			// not semver
			res = v.Version
			break
		}

		isNotPreRelease := sv.PreRelease == ""
		if isNotPreRelease {
			res = sv.String()
			break
		}

	}

	return res, nil
}

func (c Chart) Push(registry string) (string, error) {

	settings := cli.New()

	HelmDriver := "configmap"
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), HelmDriver, slog.Info); err != nil {
		slog.Error(fmt.Sprintf("%+v", err))
		return "", err
	}

	path, err := c.pullTar()
	if err != nil {
		return "", err
	}

	defer os.Remove(path)

	opts := []action.PushOpt{
		action.WithPushConfig(actionConfig),
		// action.WithInsecureSkipTLSVerify(true),
		action.WithPlainHTTP(true),
	}
	push := action.NewPushWithOpts(opts...)
	push.Settings = settings

	return push.Run(path, registry)

}

func (c Chart) pullTar() (string, error) {

	co := action.ChartPathOptions{
		InsecureSkipTLSverify: false, // work with insecure chart registries
		RepoURL:               c.URL,
		Version:               c.Version,
	}
	settings := cli.New()

	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	// HELM_DRIVER can be one of: [ configmap, secret, sql ]
	HelmDriver := "configmap"
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), HelmDriver, log.Printf); err != nil {
		// slog.Error("%+v", err)
		return "", err
	}

	opts := []action.PullOpt{
		action.WithConfig(actionConfig),
	}
	pull := action.NewPullWithOpts(opts...)
	pull.ChartPathOptions = co
	pull.Settings = settings

	dir := os.TempDir()
	pull.DestDir = dir

	_, err := pull.Run(c.Name)
	if err != nil {
		return "", err
	}

	// resolve filepath (wildcards)
	matches, err := filepath.Glob(fmt.Sprintf("%s/%s-%s.tgz", dir, c.Name, c.Version))
	if err != nil {
		return "", err
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i] < matches[j]
	})

	return matches[0], nil
}

func (c Chart) Pull() (string, error) {

	co := action.ChartPathOptions{
		InsecureSkipTLSverify: false, // work with insecure chart registries
		RepoURL:               c.URL,
		Version:               c.Version,
	}
	settings := cli.New()

	// check if artifact already exists
	tarPath := fmt.Sprintf("%s/%s-%s.tgz", settings.EnvVars()["HELM_CACHE_HOME"], c.Name, c.Version)
	chartPath := fmt.Sprintf("%s/%s", settings.EnvVars()["HELM_CACHE_HOME"], c.Name)
	if file.Exists(chartPath) &&
		file.Exists(tarPath) {
		return chartPath, nil
	}

	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	// HELM_DRIVER can be one of: [ configmap, secret, sql ]
	HelmDriver := "configmap"
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), HelmDriver, log.Printf); err != nil {
		// slog.Error("%+v", err)
		return "", err
	}

	opts := []action.PullOpt{
		action.WithConfig(actionConfig),
	}
	pull := action.NewPullWithOpts(opts...)
	pull.ChartPathOptions = co
	pull.Settings = settings
	pull.Untar = true
	pull.DestDir = settings.EnvVars()["HELM_CACHE_HOME"]

	_, err := pull.Run(c.Name)
	if err != nil {
		return "", err
	}

	return filepath.Join(pull.DestDir, c.Name), nil
}

func (c Chart) Locate() (string, error) {
	config := cli.New()

	co := action.ChartPathOptions{
		InsecureSkipTLSverify: false, // zscaler
		RepoURL:               c.URL,
		Version:               c.Version,
	}

	chartPath, err := co.LocateChart(c.Name, config)
	if err != nil {
		// subcharts nested in parent charts source?
		if c.Parent != nil {
			path := strings.Join([]string{os.Getenv("HOME"), ".cache", "helm", c.Parent.Name, "charts", c.Name}, "/")
			if file.Exists(path) {
				return path, nil
			}
			// slog.Error(fmt.Sprintf("%+v", err))
		}

		ma := downloader.Manager{
			ChartPath:  chartPath,
			SkipUpdate: false,
		}
		err := ma.Update()
		if err != nil {
			return "", err
		}

		chartPath, err := co.LocateChart(c.Name, config)
		if err == nil {
			return chartPath, nil
		}
	}

	return chartPath, nil
}

func (c Chart) Values() (map[string]any, error) {

	// Get remote Helm Chart using Helm SDK
	path, err := c.Locate()
	if err != nil {
		return nil, err
	}

	// Get detailed information about the chart
	chartRef, err := loader.Load(path)
	if err != nil {
		return nil, err
	}

	var values chartutil.Values
	// check if file exists, or use default values
	if file.Exists(c.ValuesFilePath) {
		valuesFromFile, err := chartutil.ReadValuesFile(c.ValuesFilePath)
		if err != nil {
			return nil, err
		}
		values = valuesFromFile.AsMap()
	} else {
		values = chartRef.Values
	}

	vs, err := chartutil.CoalesceValues(chartRef, values)
	if err != nil {
		return nil, err
	}

	return vs.AsMap(), nil
}

func (c *Chart) Read(update bool) (string, *chart.Chart, map[string]any, error) {

	// Check for latest version of chart
	if update {
		latest, err := c.LatestVersion()
		if err != nil {
			return "", nil, nil, err
		}
		c.Version = latest
	}

	// Get remote Helm Chart using Helm SDK
	path, err := c.Locate()
	if err != nil {
		return "", nil, nil, err
	}

	// Get detailed information about the chart
	chartRef, err := loader.Load(path)
	if err != nil {
		return "", nil, nil, err
	}

	// Get custom Helm values
	values, err := c.Values()
	if err != nil {
		return "", nil, nil, err
	}

	return path, chartRef, values, nil
}
