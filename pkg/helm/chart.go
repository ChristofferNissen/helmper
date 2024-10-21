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
func (c Chart) addToHelmRepositoryFile() (bool, error) {
	config := cli.New()
	repoConfig := config.RepositoryConfig

	var f *repo.File = repo.NewFile()
	if file.Exists(repoConfig) {
		file, err := repo.LoadFile(repoConfig)
		if err != nil {
			return false, err
		}
		f = file
	}

	if !f.Has(c.Repo.Name) {
		f.Update(&c.Repo)
		return true, f.WriteFile(repoConfig, 0644)
	}

	return false, nil
}

func (c Chart) CountDependencies() (int, error) {

	settings := cli.New()

	HelmDriver := "configmap"
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), HelmDriver, slog.Info); err != nil {
		slog.Error(fmt.Sprintf("%+v", err))
		return 0, err
	}

	path, err := c.Locate()
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

func (c Chart) Push(registry string, insecure bool, plainHTTP bool) (string, error) {
	chartFilePath, err := c.Locate()
	if err != nil {
		return "", fmt.Errorf("failed to pull tar: %w", err)
	}
	defer os.Remove(chartFilePath)

	err = c.push(chartFilePath, fmt.Sprintf("%s/charts/%s:%s", registry, c.Name, c.Version))
	return chartFilePath, err
}

func (c *Chart) modifyRegistryReferences(newRegistry string) (string, error) {
	chartFilePath, err := c.Locate()
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
				v, err := chart.ResolveVersion()
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
	ma := getManager(&buf, true, true)
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

func (c Chart) PushAndModify(registry string, insecure bool, plainHTTP bool) (string, error) {
	// Modify chart
	modifiedPath, err := c.modifyRegistryReferences(registry)
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

func (c Chart) Pull() (string, error) {
	u, err := url.Parse(c.Repo.URL)
	if err != nil {
		return "", err
	}

	settings := cli.New()
	helmCacheHome := settings.EnvVars()["HELM_CACHE_HOME"]
	tarPath := fmt.Sprintf("%s/%s-%s.tgz", helmCacheHome, c.Name, c.Version)
	chartPath := fmt.Sprintf("%s/%s", helmCacheHome, c.Name)

	if file.FileExists(chartPath) {
		return chartPath, nil
	}

	if file.FileExists(tarPath) {
		return tarPath, nil
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

		err = file.Write(tarPath, p.Chart.Data)
		if err != nil {
			return "", err
		}

	} else {
		// // Make temporary folder for tar archives
		// f, err := os.MkdirTemp(os.TempDir(), "untar")
		// if err != nil {
		// 	return "", err
		// }
		// defer os.RemoveAll(f)

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
		pull.UntarDir = chartPath
		pull.DestDir = helmCacheHome

		if _, err := pull.Run(c.Name); err != nil {
			return "", err
		}

		return chartPath, nil
	}

	return tarPath, nil
}

func (c Chart) Locate() (string, error) {
	settings := cli.New()
	helmCacheHome := settings.EnvVars()["HELM_CACHE_HOME"]

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

		return fmt.Sprintf("%s/%s-%s.tgz", helmCacheHome, c.Name, c.Version), nil
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

func (c Chart) Values() (map[string]any, error) {
	// Get detailed information about the chart
	chartRef, err := c.ChartRef()
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
		pv, err := c.Parent.Values()
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

func (c *Chart) ChartRef() (*chart.Chart, error) {
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
	return chartRef, nil
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

	// Get detailed information about the chart
	chartRef, err := c.ChartRef()
	if err != nil {
		return "", nil, nil, err
	}

	// Get custom Helm values
	values, err := c.Values()
	if err != nil {
		return "", nil, nil, err
	}

	return chartRef.ChartFullPath(), chartRef, values, nil
}
