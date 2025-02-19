package helm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChristofferNissen/helmper/pkg/util/file"
	"gopkg.in/yaml.v3"

	"helm.sh/helm/v3/pkg/registry"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

func DependencyToChart(d *chart.Dependency, p *Chart) *Chart {
	// Backwards compatibility with Charts pushed with helmper 0.1.x
	if strings.HasPrefix(d.Repository, "oci://") {
		if !strings.HasSuffix(d.Repository, d.Name) {
			if strings.HasSuffix(d.Repository, "/charts") {
				d.Repository = d.Repository + "/" + d.Name
			} else {
				d.Repository = d.Repository + "/charts/" + d.Name
			}
		}
	}

	return &Chart{
		Name: d.Name,
		Repo: repo.Entry{
			Name: p.Repo.Name + "/" + d.Name,
			URL:  d.Repository,
		},
		Version:        d.Version,
		Parent:         p,
		ValuesFilePath: p.ValuesFilePath,
		Values:         p.Values,
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
		f.WriteFile(settings.RepositoryConfig, 0o644)
	}

	if !f.Has(c.Repo.Name) {
		f.Update(&c.Repo)
		return true, f.WriteFile(settings.RepositoryConfig, 0o644)
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

	dest, _ := strings.CutPrefix(destination, "oci://")
	_, err = c.RegistryClient.Push(bs, dest)
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

	err = c.push(chartFilePath, fmt.Sprintf("%s/%s/%s:%s", registry, chartutil.ChartsDir, c.Name, c.Version))
	return chartFilePath, err
}

func (c *Chart) modifyRegistryReferences(settings *cli.EnvSettings, newRegistry string, prefixSource bool) (string, error) {
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

	// Modify dependencies
	for _, d := range chartRef.Metadata.Dependencies {
		switch {
		case strings.HasPrefix(d.Repository, "file://"):
			d.Repository = ""
		case d.Repository != "":
			// Change dependency ref to registry being imported to
			d.Repository = newRegistry + "/charts/" + d.Name

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
	// Write modified dependencies to Chart.yaml
	err = chartutil.SaveChartfile(filepath.Join(dir, c.Name, chartutil.ChartfileName), chartRef.Metadata)
	if err != nil {
		return "", err
	}

	// Remove Lock file
	err = removeLockFile(filepath.Join(dir, c.Name))
	if err != nil {
		return "", err
	}

	// Reload Chart from filesystem
	chartRef, err = loader.Load(filepath.Join(dir, c.Name))
	if err != nil {
		return "", err
	}

	// Replace Image References in values.yaml
	replaceImageReferences(chartRef.Values, newRegistry, prefixSource)
	for _, r := range chartRef.Raw {
		if r.Name == "values.yaml" {
			d, err := yaml.Marshal(chartRef.Values)
			if err != nil {
				return "", err
			}
			r.Data = d
		}
	}

	// Compute the SHA256 digest of the chart metadata
	metadataBytes, err := yaml.Marshal(chartRef.Metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal chart metadata: %w", err)
	}
	sha := sha256.Sum256(metadataBytes)
	digest := hex.EncodeToString(sha[:])

	// Create the Chart.lock content
	lock := chart.Lock{
		Generated:    time.Now(),
		Dependencies: chartRef.Metadata.Dependencies,
		Digest:       digest,
	}

	// Serialize the lock file content
	data, err := yaml.Marshal(&lock)
	if err != nil {
		return "", fmt.Errorf("failed to marshal lock file: %w", err)
	}

	// Write the lock file to the chart path
	lockFilePath := filepath.Join(dir, c.Name, "Chart.lock")
	if err := os.WriteFile(lockFilePath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write lock file: %w", err)
	}

	// Validate Chart
	b, err := chartutil.IsChartDir(filepath.Join(dir, c.Name))
	if err != nil {
		return "", fmt.Errorf("could not validate chart directory")
	}
	if b {
		// Save modified Helm Chart to filesystem before push
		modifiedPath, err := chartutil.Save(chartRef, "/tmp/")
		if err != nil {
			return "", err
		}
		return modifiedPath, nil
	}

	return "", fmt.Errorf("modified chart is malformed")
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

func (c Chart) PushAndModify(settings *cli.EnvSettings, registry string, insecure bool, plainHTTP bool, prefixSource bool) (string, error) {
	// Modify chart
	modifiedPath, err := c.modifyRegistryReferences(settings, registry, prefixSource)
	if err != nil {
		return "", err
	}

	// Use the `Push` method to push the modified chart
	c.PlainHTTP = plainHTTP
	c.Repo.InsecureSkipTLSverify = insecure
	err = c.push(modifiedPath, fmt.Sprintf("%s/%s/%s:%s", registry, chartutil.ChartsDir, c.Name, c.Version))
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
		path := filepath.Join(chartPath, chartutil.ChartsDir, c.Name, chartutil.ChartfileName)
		if file.FileExists(path) {
			// Check chart is the correct version
			meta, err := chartutil.LoadChartfile(path)
			if err != nil {
				return "", err
			}

			if meta.Version == c.Version {
				slog.Info("Reusing existing achieve for chart", slog.String("chart", c.Name), slog.String("path", chartPath))
				return chartPath, nil
			}

			slog.Info("Deleting existing achieve for chart", slog.String("chart", c.Name), slog.String("desired version", c.Version), slog.String("found version", meta.Version), slog.String("path", chartPath))
			err = os.RemoveAll(chartPath)
			if err != nil {
				return "", err
			}
		}
	}

	if foundPath, ok := findFile(tarPattern); ok {
		slog.Info("Reusing existing tar archieve for chart", slog.String("chart", c.Name), slog.String("path", foundPath))
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

		opts := []registry.ClientOption{}
		if c.PlainHTTP {
			opts = append(opts, registry.ClientOptPlainHTTP())
		}
		rc, err := registry.NewClient(
			opts...,
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
		pull.UntarDir = chartutil.ChartsDir

		if _, err := pull.Run(c.Name); err != nil {
			return "", err
		}

		f, b := findFile(fmt.Sprintf("%s/%s/%s-*%s*.tgz", chartPath, chartutil.ChartsDir, c.Name, c.Version))
		if b {
			os.RemoveAll(f)
		}

		return chartPath + "/" + chartutil.ChartsDir + "/" + c.Name, nil
	}
}

func (c Chart) Locate(settings *cli.EnvSettings) (string, error) {
	// Check if the repository URL is an OCI URL
	if strings.HasPrefix(c.Repo.URL, "oci://") {
		// Pull the chart from OCI

		return c.Pull(settings)
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

func (c Chart) GetValues(settings *cli.EnvSettings) (map[string]any, error) {
	// Get detailed information about the chart
	chartRef, err := c.ChartRef(settings)
	if err != nil {
		return nil, err
	}

	// Check if file exists, or use default values
	var values chartutil.Values = chartRef.Values
	if file.Exists(c.ValuesFilePath) {
		valuesFromFile, err := chartutil.ReadValuesFile(c.ValuesFilePath)
		if err != nil {
			return nil, err
		}
		values = valuesFromFile.AsMap()
	}

	vs, err := chartutil.CoalesceValues(chartRef, values)
	if err != nil {
		return nil, err
	}

	if c.Parent != nil {
		pv, err := c.Parent.GetValues(settings)
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
	values, err := c.GetValues(settings)
	if err != nil {
		return "", nil, nil, err
	}

	return chartRef.ChartFullPath(), chartRef, values, nil
}
