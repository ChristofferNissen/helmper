package helm

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChristofferNissen/helmper/pkg/util/file"
	"gopkg.in/yaml.v3"

	"helm.sh/helm/v3/pkg/registry"
	helm_registry "helm.sh/helm/v3/pkg/registry"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

func DependencyToChart(d *chart.Dependency, p *Chart) *Chart {
	return &Chart{
		Name: d.Name,
		Repo: repo.Entry{
			Name: p.Repo.Name + "/" + d.Name,
			URL:  d.Repository,
		},
		Version:        d.Version,
		Parent:         p,
		ValuesFilePath: p.ValuesFilePath,
		DepsCount:      0,
		PlainHTTP:      p.PlainHTTP,
		RegistryClient: p.RegistryClient,
		IndexFileLoader: &FunctionLoader{
			LoadFunc: repo.LoadIndexFile,
		},
	}
}

// addToHelmRepositoryFile adds repository to Helm repository.yml to enable querying/pull
func (c Chart) addToHelmRepositoryFile(settings *cli.EnvSettings) (bool, error) {

	var f *repo.File = repo.NewFile()
	if file.Exists(settings.RepositoryConfig) {
		file, err := repo.LoadFile(settings.RepositoryConfig)
		if err != nil {
			return false, err
		}
		f = file
	} else {
		f.WriteFile(settings.RepositoryConfig, 0644)
	}

	if !f.Has(c.Repo.Name) {
		f.Update(&c.Repo)
		return true, f.WriteFile(settings.RepositoryConfig, 0644)
	}

	return false, nil
}

func (c Chart) CountDependencies(settings *cli.EnvSettings) (int, error) {

	HelmDriver := "configmap"
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), HelmDriver, slog.Info); err != nil {
		slog.Error(fmt.Sprintf("%+v", err))
		return 0, err
	}

	path, err := c.Locate(settings)
	if err != nil {
		return 0, err
	}
	defer os.Remove(path)

	chartRef, err := loader.Load(path)
	if err != nil {
		return 0, err
	}

	return len(chartRef.Metadata.Dependencies), nil
}

func (c Chart) push(chartFilePath string, destination string) error {
	bs, err := os.ReadFile(chartFilePath)
	if err != nil {
		return fmt.Errorf("error reading chart: %w", err)
	}

	_, err = c.RegistryClient.Push(bs, destination)
	if err != nil {
		return fmt.Errorf("error pushing chart: %w", err)
	}

	log.Printf("Chart pushed successfully: %s to %s\n", c.Name, destination)
	return nil
}

func (c Chart) Push(settings *cli.EnvSettings, registry string, insecure bool, plainHTTP bool) (string, error) {
	chartFilePath, err := c.Locate(settings)
	if err != nil {
		return "", fmt.Errorf("failed to pull tar: %w", err)
	}
	defer os.Remove(chartFilePath)

	err = c.push(chartFilePath, fmt.Sprintf("%s/charts/%s:%s", registry, c.Name, c.Version))
	return chartFilePath, err
}

func (c *Chart) modifyRegistryReferences(settings *cli.EnvSettings, newRegistry string) (string, error) {
	chartFilePath, err := c.Locate(settings)
	if err != nil {
		return "", err
	}
	defer os.Remove(chartFilePath)

	// Create a temporary directory for modification
	dir, err := os.MkdirTemp("", "sampledir")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	// Expand the chart for modification
	err = chartutil.ExpandFile(dir, chartFilePath)
	if err != nil {
		return "", err
	}

	// Modify chart contents before pushing
	chartRef, err := loader.Load(filepath.Join(dir, c.Name))
	if err != nil {
		return "", err
	}

	// Dependencies (Chart.yaml)
	for _, d := range chartRef.Metadata.Dependencies {
		switch {
		case strings.HasPrefix(d.Repository, "file://"):
			// d.Repository = filepath.Join(dir, c.Name, "charts", d.Name)
			d.Repository = ""
		case d.Repository != "":

			// Change dependency ref to registry being imported to
			d.Repository = newRegistry

			if strings.Contains(d.Version, "*") || strings.Contains(d.Version, "x") {
				chart := DependencyToChart(d, c)

				// OCI dependencies can not use globs in version
				// Resolve Globs to latest patch
				v, err := chart.ResolveVersion(settings)
				if err == nil {
					d.Version = v
				}
			}
		}
	}

	err = chartutil.SaveChartfile(filepath.Join(dir, c.Name, "Chart.yaml"), chartRef.Metadata)
	if err != nil {
		return "", err
	}

	// Remove Lock file
	err = removeLockFile(filepath.Join(dir, c.Name))
	if err != nil {
		return "", err
	}

	// Helm Dependency Update
	var buf bytes.Buffer
	ma := getManager(settings, &buf, true, true)
	ma.ChartPath = filepath.Join(dir, c.Name)
	err = ma.Update()
	if err != nil {
		slog.Info(buf.String())
		log.Printf("Error occurred trying to update Helm Chart on filesystem: %v, skipping update of chart dependencies", err)
	}

	// Reload Helm Chart from filesystem
	chartRef, err = loader.Load(filepath.Join(dir, c.Name))
	if err != nil {
		return "", err
	}

	// Replace Image References in values.yaml
	replaceImageReferences(chartRef.Values, newRegistry)
	for _, r := range chartRef.Raw {
		if r.Name == "values.yaml" {
			d, err := yaml.Marshal(chartRef.Values)
			if err != nil {
				return "", err
			}
			r.Data = d
		}
	}

	// Save modified Helm Chart to filesystem before push
	modifiedPath, err := chartutil.Save(chartRef, "/tmp/")
	if err != nil {
		return "", err
	}

	return modifiedPath, nil
}

// Function to remove the lock file
func removeLockFile(chartPath string) error {

	// Locate the lock file
	lockFilePath := filepath.Join(chartPath, "Chart.lock")
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		// return fmt.Errorf("lock file does not exist")
		return nil
	}

	// Remove the lock file
	if err := os.Remove(lockFilePath); err != nil {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	slog.Debug("Lock file removed successfully")
	return nil
}

func (c Chart) PushAndModify(settings *cli.EnvSettings, registry string, insecure bool, plainHTTP bool) (string, error) {
	// Modify chart
	modifiedPath, err := c.modifyRegistryReferences(settings, registry)
	if err != nil {
		return "", err
	}

	// Use the `Push` method to push the modified chart
	c.PlainHTTP = plainHTTP
	c.Repo.InsecureSkipTLSverify = insecure
	err = c.push(modifiedPath, fmt.Sprintf("%s/charts/%s:%s", registry, c.Name, c.Version))
	if err != nil {
		return "", err
	}

	return modifiedPath, nil
}

func findFile(pattern string) (string, bool) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Printf("Error matching pattern: %v\n", err)
		return "", false
	}
	if len(matches) > 0 {
		return matches[0], true
	}
	return "", false
}

func (c Chart) Pull(settings *cli.EnvSettings) (string, error) {
	u, err := url.Parse(c.Repo.URL)
	if err != nil {
		return "", err
	}

	chartPath := strings.Replace(settings.RepositoryCache, "/repository", "/"+c.Name, 1)
	tarPattern := fmt.Sprintf("%s-*%s*.tgz", chartPath, c.Version)

	if file.FileExists(chartPath) {
		return chartPath, nil
	}

	if foundPath, ok := findFile(tarPattern); ok {
		return foundPath, nil
	}

	ref := func() string {
		url, _ := strings.CutPrefix(c.Repo.URL, "oci://")
		if strings.HasSuffix(url, c.Name) {
			return url + ":" + c.Version
		} else {
			return url + "/" + c.Name + ":" + c.Version
		}
	}()

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		if err := os.Setenv("HELM_EXPERIMENTAL_OCI", "1"); err != nil {
			return "", err
		}

		rc, err := registry.NewClient(
			registry.ClientOptDebug(true),
			registry.ClientOptPlainHTTP(),
			// registry.ClientOptInsecureSkipVerifyTLS(c.Repo.InsecureSkipTLSverify),
		)
		if err != nil {
			return "", err
		}

		p, err := rc.Pull(ref)
		if err != nil {
			return "", err
		}

		tarPath := fmt.Sprintf("%s-%s.tgz", chartPath, c.Version)

		err = file.Write(tarPath, p.Chart.Data)
		if err != nil {
			return "", err
		}

		return tarPath, nil

	} else {

		co := action.ChartPathOptions{
			CaFile:                c.Repo.CAFile,
			CertFile:              c.Repo.CertFile,
			KeyFile:               c.Repo.KeyFile,
			InsecureSkipTLSverify: c.Repo.InsecureSkipTLSverify,
			PlainHTTP:             u.Scheme == "http",
			RepoURL:               c.Repo.URL,
			Username:              c.Repo.Username,
			Password:              c.Repo.Password,
			Version:               c.Version,
		}

		actionConfig := new(action.Configuration)
		if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "configmap", log.Printf); err != nil {
			return "", err
		}

		pull := action.NewPullWithOpts(action.WithConfig(actionConfig))
		pull.ChartPathOptions = co
		pull.Settings = settings
		pull.Untar = true
		pull.DestDir = chartPath
		pull.UntarDir = "charts"

		if _, err := pull.Run(c.Name); err != nil {
			return "", err
		}

		f, b := findFile(fmt.Sprintf("%s/charts/%s-*%s*.tgz", chartPath, c.Name, c.Version))
		if b {
			os.RemoveAll(f)
		}

		return chartPath + "/charts/" + c.Name, nil
	}
}

func (c Chart) Locate(settings *cli.EnvSettings) (string, error) {

	// Check if the repository URL is an OCI URL
	if strings.HasPrefix(c.Repo.URL, "oci://") {
		// Pull the chart from OCI
		ref := strings.TrimSuffix(c.Repo.URL, "/") + "/" + c.Name + ":" + c.Version
		if err := os.Setenv("HELM_EXPERIMENTAL_OCI", "1"); err != nil {
			return "", err
		}

		rc, err := helm_registry.NewClient()
		if err != nil {
			return "", err
		}

		if _, err := rc.Pull(ref); err != nil {
			return "", err
		}

		return fmt.Sprintf("%s-%s.tgz", settings.RepositoryCache, c.Version), nil
	}

	// For non-OCI URLs, use ChartPathOptions
	co := action.ChartPathOptions{
		CaFile:                c.Repo.CAFile,
		CertFile:              c.Repo.CertFile,
		KeyFile:               c.Repo.KeyFile,
		InsecureSkipTLSverify: c.Repo.InsecureSkipTLSverify,
		PlainHTTP:             c.Repo.PassCredentialsAll,
		RepoURL:               c.Repo.URL,
		Username:              c.Repo.Username,
		Password:              c.Repo.Password,
		Version:               c.Version,
	}

	// Locate the chart path
	chartPath, err := co.LocateChart(c.Name, settings)
	if err != nil {
		return "", err
	}

	return chartPath, nil
}

func (c Chart) Values(settings *cli.EnvSettings) (map[string]any, error) {
	// Get detailed information about the chart
	chartRef, err := c.ChartRef(settings)
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

	if c.Parent != nil {
		pv, err := c.Parent.Values(settings)
		if err != nil {
			return nil, err
		}

		vs, err = chartutil.CoalesceValues(chartRef, pv[c.Name].(map[string]interface{}))
		if err != nil {
			return nil, err
		}
	}

	return vs.AsMap(), nil
}

func (c *Chart) ChartRef(settings *cli.EnvSettings) (*chart.Chart, error) {
	// Get remote Helm Chart using Helm SDK
	path, err := c.Locate(settings)
	if err != nil {
		return nil, err
	}
	// Get detailed information about the chart
	chartRef, err := loader.Load(path)
	if err != nil {
		return nil, err
	}
	return chartRef, nil
}

func (c *Chart) Read(settings *cli.EnvSettings, update bool) (string, *chart.Chart, map[string]any, error) {
	// Check for latest version of chart
	if update {
		latest, err := c.LatestVersion(settings)
		if err != nil {
			return "", nil, nil, err
		}
		c.Version = latest
	}

	// Get detailed information about the chart
	chartRef, err := c.ChartRef(settings)
	if err != nil {
		return "", nil, nil, err
	}

	// Get custom Helm values
	values, err := c.Values(settings)
	if err != nil {
		return "", nil, nil, err
	}

	return chartRef.ChartFullPath(), chartRef, values, nil
}
