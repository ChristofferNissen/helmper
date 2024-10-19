package helm

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChristofferNissen/helmper/pkg/util/file"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"

	"helm.sh/helm/v3/pkg/registry"
	helm_registry "helm.sh/helm/v3/pkg/registry"

	"github.com/blang/semver/v4"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
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
	DepsCount      int
}

func DependencyToChart(d *chart.Dependency, p Chart) Chart {
	return Chart{
		Name: d.Name,
		Repo: repo.Entry{
			Name: p.Repo.Name + "/" + d.Name,
			URL:  d.Repository,
		},
		Version:        d.Version,
		Parent:         &p,
		ValuesFilePath: p.ValuesFilePath,
		DepsCount:      0,
		PlainHTTP:      p.PlainHTTP,
	}
}

// AddChartRepositoryToHelmRepositoryFile adds repository to Helm repository.yml to enable querying/pull
func (c Chart) AddToHelmRepositoryFile() (bool, error) {
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

func VersionsInRange(r semver.Range, c Chart) ([]string, error) {
	prefixV := strings.Contains(c.Version, "v")

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
		sv, err := semver.ParseTolerant(v.Version)
		if err != nil {
			continue
		}

		if len(sv.Pre) > 0 {
			continue
		}

		if r(sv) {
			//valid
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

	prefixV := strings.Contains(c.Version, "v")
	version := strings.ReplaceAll(c.Version, "v", "")

	r, err := semver.ParseRange(version)
	if err != nil {
		// not a semver range
		return nil, err
	}

	if strings.HasPrefix(c.Repo.URL, "oci://") {
		ref := strings.TrimPrefix(strings.TrimSuffix(c.Repo.URL, "/")+"/"+c.Name, "oci://")

		repo, err := remote.NewRepository(ref)
		if err != nil {
			return []string{}, err
		}

		repo.PlainHTTP = c.PlainHTTP

		// prepare authentication using Docker credentials
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

		vs := []semver.Version{}
		err = repo.Tags(context.TODO(), "", func(tags []string) error {
			for _, t := range tags {
				s, err := semver.ParseTolerant(t)
				if err != nil {
					// non semver tag
					continue
				}
				vs = append(vs, s)
			}

			semver.Sort(vs)

			return nil
		})
		if err != nil {
			return []string{}, err
		}

		versionsInRange := []string{}
		for _, v := range vs {
			if len(v.Pre) > 0 {
				continue
			}

			if r(v) {
				//valid
				s := v.String()
				if prefixV {
					s = "v" + s
				}
				versionsInRange = append(versionsInRange, s)
			}

		}

		return versionsInRange, nil
	}

	update, err := c.AddToHelmRepositoryFile()
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

		repo, err := remote.NewRepository(ref)
		if err != nil {
			return "", err
		}

		repo.PlainHTTP = c.PlainHTTP

		// prepare authentication using Docker credentials
		storeOpts := credentials.StoreOptions{}
		credStore, err := credentials.NewStoreFromDocker(storeOpts)
		if err != nil {
			return "", err
		}
		repo.Client = &auth.Client{
			Client:     retry.DefaultClient,
			Cache:      auth.NewCache(),
			Credential: credentials.Credential(credStore), // Use the credentials store
		}

		vs := []semver.Version{}
		err = repo.Tags(context.TODO(), "", func(tags []string) error {
			for _, t := range tags {
				s, err := semver.ParseTolerant(t)
				if err != nil {
					// non semver tag
					continue
				}

				if r(s) {
					vs = append(vs, s)
				}
			}

			semver.Sort(vs)

			return nil
		})
		if err != nil {
			return "", err
		}

		if len(vs) > 0 {
			return vs[len(vs)-1].String(), nil
		}

		return "", xerrors.Errorf("Not found")
	}

	update, err := c.AddToHelmRepositoryFile()
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
		switch {
		case err != nil:
			// not semver
			continue
		case len(sv.Pre) > 0:
			continue
		case r(sv):
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

		// prepare authentication using Docker credentials
		storeOpts := credentials.StoreOptions{}
		credStore, err := credentials.NewStoreFromDocker(storeOpts)
		if err != nil {
			return "", err
		}
		repo.Client = &auth.Client{
			Client:     retry.DefaultClient,
			Cache:      auth.NewCache(),
			Credential: credentials.Credential(credStore), // Use the credentials store
		}

		vPrefix := strings.Contains(c.Version, "v")
		l := c.Version
		err = repo.Tags(context.TODO(), "", func(tags []string) error {
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
			l = vs[len(vs)-1].String()

			if vPrefix {
				l = "v" + l
			}

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

// func untar(tarPath string) (string, error) {
// 	tempDir, err := os.MkdirTemp("", "untar")
// 	if err != nil {
// 		return "", err
// 	}

// 	file, err := os.Open(tarPath)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer file.Close()

// 	gzipReader, err := gzip.NewReader(file)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer gzipReader.Close()

// 	tarReader := tar.NewReader(gzipReader)
// 	for {
// 		header, err := tarReader.Next()
// 		if err == io.EOF {
// 			break
// 		}
// 		if err != nil {
// 			return "", err
// 		}

// 		targetPath := filepath.Join(tempDir, header.Name)
// 		switch header.Typeflag {
// 		case tar.TypeDir:
// 			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
// 				return "", err
// 			}
// 		case tar.TypeReg:
// 			outFile, err := os.Create(targetPath)
// 			if err != nil {
// 				return "", err
// 			}
// 			if _, err := io.Copy(outFile, tarReader); err != nil {
// 				outFile.Close()
// 				return "", err
// 			}
// 			outFile.Close()
// 		default:
// 			return "", fmt.Errorf("unknown type: %b in %s", header.Typeflag, header.Name)
// 		}
// 	}

// 	return tempDir, nil
// }

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

func readFileAsBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c Chart) push(chartFilePath string, destination string) error {
	// Read chart bytes
	bs, err := readFileAsBytes(chartFilePath)
	if err != nil {
		log.Fatalf("Error reading chart: %v", err)
	}

	// Create a new registry client
	rc, err := helm_registry.NewClient(
		helm_registry.ClientOptDebug(true),
		helm_registry.ClientOptPlainHTTP(),
	)
	if err != nil {
		log.Fatalf("Error creating registry client: %v", err)
	}
	_, err = rc.Push(
		bs,
		destination,
		helm_registry.PushOptStrictMode(false),
	)
	if err != nil {
		log.Fatalf("Error pushing chart: %v", err)
	}

	fmt.Println("Chart pushed successfully")

	return nil
}

func (c Chart) Push(registry string, insecure bool, plainHTTP bool) (string, error) {
	chartFilePath, err := c.Pull()
	if err != nil {
		return "", fmt.Errorf("failed to pull tar: %w", err)
	}
	defer os.Remove(chartFilePath)

	err = c.push(chartFilePath, fmt.Sprintf("%s/charts/%s:%s", registry, c.Name, c.Version))
	return chartFilePath, err
}

func (c Chart) PushAndModify(registry string, insecure bool, plainHTTP bool) (string, error) {
	chartFilePath, err := c.Pull()
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

	// Update Dependencies in Chart.yaml
	for _, d := range chartRef.Metadata.Dependencies {
		if !strings.HasPrefix(d.Repository, "file://") && d.Repository != "" {
			d.Repository = registry
			if strings.Contains(d.Version, "*") || strings.Contains(d.Version, "x") {
				v, err := c.ResolveVersion()
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

	// Helm Dependency Update
	var buf bytes.Buffer
	ma := getManager(&buf, true, true)
	ma.ChartPath = filepath.Join(dir, c.Name)
	err = ma.Update()
	if err != nil {
		log.Printf("Error occurred trying to update Helm Chart on filesystem: %v, skipping update of chart dependencies", err)
	}

	// Reload Helm Chart from filesystem
	chartRef, err = loader.Load(filepath.Join(dir, c.Name))
	if err != nil {
		return "", err
	}

	// Replace Image References in values.yaml
	replaceImageReferences(chartRef.Values, registry)
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

	// Use the `Push` method to push the modified chart
	c.PlainHTTP = plainHTTP
	c.Repo.InsecureSkipTLSverify = insecure
	err = c.push(modifiedPath, fmt.Sprintf("%s/charts/%s:%s", registry, c.Name, c.Version))
	if err != nil {
		return "", err
	}

	return modifiedPath, nil
}

func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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

	if fileExists(chartPath) && fileExists(tarPath) {
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
		pull.Untar = false
		pull.DestDir = helmCacheHome

		if _, err := pull.Run(c.Name); err != nil {
			return "", err
		}
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
