package helm

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ChristofferNissen/helmper/pkg/util/file"
	"golang.org/x/xerrors"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/blang/semver/v4"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
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
	Name           string     `json:"name"`
	Version        string     `json:"version"`
	ValuesFilePath string     `json:"valuesFilePath"`
	Repo           repo.Entry `json:"repo"`
	Parent         *Chart
	Images         *Images `json:"images"`
	PlainHTTP      bool    `json:"plainHTTP"`
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

	if !f.Has(c.Repo.Name) {
		f.Update(&c.Repo)
		return f.WriteFile(repoConfig, 0644)
	}

	return nil
}

func (c Chart) ResolveVersions() ([]string, error) {

	r, err := semver.ParseRange(c.Version)
	if err != nil {
		// not a semver range
		return nil, err
	}

	// Fetch versions from Helm Repository
	config := cli.New()
	indexPath := fmt.Sprintf("%s/%s-index.yaml", config.RepositoryCache, c.Repo.Name)
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return nil, err
	}
	index.SortEntries()
	versions := index.Entries[c.Name]

	versionsInRange := []string{}
	for _, v := range versions {

		sv, err := semver.Parse(v.Version)
		if err != nil {
			continue
		}

		if len(sv.Pre) > 0 {
			continue
		}

		if r(sv) {
			//valid
			versionsInRange = append(versionsInRange, sv.String())
		}

	}

	return versionsInRange, nil
}

func (c Chart) ResolveVersion() (string, error) {

	v := strings.ReplaceAll(c.Version, "*", "0")

	s, err := semver.Parse(strings.TrimPrefix(v, "v"))
	if err != nil {
		return "", err
	}

	major := s.Major
	minor := s.Minor

	config := cli.New()
	indexPath := fmt.Sprintf("%s/%s-index.yaml", config.RepositoryCache, c.Repo.Name)
	index, err := repo.LoadIndexFile(indexPath)
	if err != nil {
		return "", err
	}
	index.SortEntries()
	versions := index.Entries[c.Name]

	for _, v := range versions {
		sv, err := semver.Parse(v.Version)
		switch {
		case err != nil:
			// not semver
			continue
		case len(sv.Pre) > 0:
			continue
		case sv.Major == major && sv.Minor == minor:
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

		l := c.Version
		err = repo.Tags(context.TODO(), c.Version, func(tags []string) error {
			vs := []semver.Version{}

			for _, t := range tags {
				s, err := semver.Parse(t)
				if err != nil {
					// non semver tag
					continue
				}
				vs = append(vs, s)
			}

			semver.Sort(vs)
			l = vs[len(vs)-1].String()

			return nil
		})
		if err != nil {
			return "", err
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
			// not semver
			res = v.Version
			break
		}

		isNotPreRelease := len(sv.Pre) == 0
		if isNotPreRelease {
			res = sv.String()
			break
		}

	}

	return res, nil
}

func (c Chart) pullTar() (string, error) {

	u, err := url.Parse(c.Repo.URL)
	if err != nil {
		return "", err
	}

	co := action.ChartPathOptions{
		CaFile:                c.Repo.CAFile,
		CertFile:              c.Repo.CertFile,
		KeyFile:               c.Repo.KeyFile,
		InsecureSkipTLSverify: c.Repo.InsecureSkipTLSverify,
		PlainHTTP:             c.PlainHTTP || u.Scheme == "https",
		Password:              c.Repo.Password,
		PassCredentialsAll:    c.Repo.PassCredentialsAll,
		RepoURL:               c.Repo.URL,
		Username:              c.Repo.Username,
		Version:               c.Version,
	}
	settings := cli.New()

	// You can pass an empty string instead of settings.Namespace() to list
	// all namespacesargo-cd
	// HELM_DRIVER can be one of: [ configmap, secret, sql ]
	HelmDriver := "configmap"
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), HelmDriver, log.Printf); err != nil {
		return "", err
	}

	opts := []action.PullOpt{
		action.WithConfig(actionConfig),
	}
	pull := action.NewPullWithOpts(opts...)
	pull.ChartPathOptions = co
	pull.Settings = settings
	tmp := os.TempDir()
	pull.DestDir = tmp

	_, err = pull.Run(c.Name)
	if err != nil {
		return "", err
	}

	// Resolve filepath (wildcards) for dependency charts
	matches, err := filepath.Glob(fmt.Sprintf("%s/%s-%s.tgz", tmp, c.Name, c.Version))
	if err != nil {
		return "", err
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i] < matches[j]
	})

	return matches[0], nil
}

func (c Chart) Push(registry string, insecure bool, plainHTTP bool) (string, error) {

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
		action.WithInsecureSkipTLSVerify(insecure),
		action.WithPlainHTTP(plainHTTP),
	}
	push := action.NewPushWithOpts(opts...)
	push.Settings = settings

	return push.Run(path, registry)
}

// const (
// 	helmChartMetaMediaType    = "application/vnd.cncf.helm.chart.meta.v1+json"
// 	helmChartContentMediaType = "application/vnd.cncf.helm.chart.content.v1+tar"
// )

func (c Chart) Pull() (string, error) {

	u, err := url.Parse(c.Repo.URL)
	if err != nil {
		return "", err
	}

	co := action.ChartPathOptions{
		CaFile:                c.Repo.CAFile,
		CertFile:              c.Repo.CertFile,
		KeyFile:               c.Repo.KeyFile,
		InsecureSkipTLSverify: c.Repo.InsecureSkipTLSverify,
		PlainHTTP:             u.Scheme == "https",
		Password:              c.Repo.Password,
		PassCredentialsAll:    c.Repo.PassCredentialsAll,
		RepoURL:               c.Repo.URL,
		Username:              c.Repo.Username,
		Version:               c.Version,
	}
	settings := cli.New()

	helmCacheHome := settings.EnvVars()["HELM_CACHE_HOME"]

	// check if artifact already exists
	tarPath := fmt.Sprintf("%s/%s-%s.tgz", helmCacheHome, c.Name, c.Version)
	chartPath := fmt.Sprintf("%s/%s", helmCacheHome, c.Name)
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
		return "", err
	}

	// Make temporary folder for tar archives
	f, err := os.MkdirTemp(os.TempDir(), "untar")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(f)

	opts := []action.PullOpt{
		action.WithConfig(actionConfig),
	}
	pull := action.NewPullWithOpts(opts...)
	pull.ChartPathOptions = co
	pull.Settings = settings
	pull.Untar = true
	pull.UntarDir = f
	pull.DestDir = helmCacheHome

	_, err = pull.Run(c.Name)
	if err != nil {
		return "", err
	}

	return filepath.Join(pull.DestDir, c.Name), nil
}

func (c Chart) Locate() (string, error) {
	config := cli.New()
	helmCacheHome := config.EnvVars()["HELM_CACHE_HOME"]

	switch {
	case strings.HasPrefix(c.Repo.URL, "oci://"):

		ref := c.Repo.URL + c.Name

		co := action.ChartPathOptions{
			CaFile:                c.Repo.CAFile,
			CertFile:              c.Repo.CertFile,
			KeyFile:               c.Repo.KeyFile,
			InsecureSkipTLSverify: c.Repo.InsecureSkipTLSverify,
			PassCredentialsAll:    c.Repo.PassCredentialsAll,
			Username:              c.Repo.Username,
			Password:              c.Repo.Password,
			Version:               c.Version,
		}

		settings := cli.New()

		// You can pass an empty string instead of settings.Namespace() to list
		// all namespaces
		// HELM_DRIVER can be one of: [ configmap, secret, sql ]
		HelmDriver := "configmap"
		actionConfig := new(action.Configuration)
		if err := actionConfig.Init(
			settings.RESTClientGetter(),
			settings.Namespace(),
			HelmDriver,
			log.Printf,
		); err != nil {
			return "", err
		}

		// Make temporary folder for tar archives
		f, err := os.MkdirTemp(os.TempDir(), "untar")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(f)

		opts := []action.PullOpt{
			action.WithConfig(actionConfig),
		}
		pull := action.NewPullWithOpts(opts...)
		pull.ChartPathOptions = co
		pull.Settings = settings
		pull.DestDir = helmCacheHome

		_, err = pull.Run(ref)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("%s/%s-%s.tgz", helmCacheHome, c.Name, c.Version), nil
	default:

		u, err := url.Parse(c.Repo.URL)
		if err != nil {
			return "", err
		}

		co := action.ChartPathOptions{
			CaFile:                c.Repo.CAFile,
			CertFile:              c.Repo.CertFile,
			KeyFile:               c.Repo.KeyFile,
			InsecureSkipTLSverify: c.Repo.InsecureSkipTLSverify,
			PlainHTTP:             u.Scheme == "https",
			Password:              c.Repo.Password,
			PassCredentialsAll:    c.Repo.PassCredentialsAll,
			RepoURL:               c.Repo.URL,
			Username:              c.Repo.Username,
			Version:               c.Version,
		}

		chartPath, err := co.LocateChart(c.Name, config)
		if err != nil {
			// subcharts nested in parent charts source?
			if c.Parent != nil {
				path := filepath.Join(helmCacheHome, c.Parent.Name, "charts", c.Name)
				if file.Exists(path) {
					return path, nil
				}
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

		// "/home/cn/.cache/helm/repository/argo-cd-5.51.6.tgz"
		return chartPath, nil
	}
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

	// Check if file exists, or use default values
	var values chartutil.Values
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
