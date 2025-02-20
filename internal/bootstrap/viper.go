package bootstrap

import (
	"log/slog"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/util/file"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
)

type ImportConfigSection struct {
	Import struct {
		Enabled                   bool    `yaml:"enabled"`
		Architecture              *string `yaml:"architecture"`
		ReplaceRegistryReferences bool    `yaml:"replaceRegistryReferences"`
		Copacetic                 struct {
			Enabled      bool `yaml:"enabled"`
			IgnoreErrors bool `yaml:"ignoreErrors"`
			Buildkitd    struct {
				Addr       string `yaml:"addr"`
				CACertPath string `yaml:"CACertPath"`
				CertPath   string `yaml:"certPath"`
				KeyPath    string `yaml:"keyPath"`
			} `yaml:"buildkitd"`
			Trivy struct {
				Addr          string `yaml:"addr"`
				Insecure      bool   `yaml:"insecure"`
				IgnoreUnfixed bool   `yaml:"ignoreUnfixed"`
			} `yaml:"trivy"`
			Output struct {
				Tars struct {
					Clean  bool   `yaml:"clean"`
					Folder string `yaml:"folder"`
				} `yaml:"tars"`
				Reports struct {
					Clean  bool   `yaml:"clean"`
					Folder string `yaml:"folder"`
				} `yaml:"reports"`
			} `yaml:"output"`
		} `yaml:"copacetic"`
		Cosign struct {
			Enabled           bool    `yaml:"enabled"`
			VerifyExisting    bool    `yaml:"verifyExisting"`
			KeyRef            string  `yaml:"keyRef"`
			KeyRefPass        *string `yaml:"keyRefPass"`
			PubKeyRef         *string `yaml:"pubKeyRef"`
			AllowHTTPRegistry bool    `yaml:"allowHTTPRegistry"`
			AllowInsecure     bool    `yaml:"allowInsecure"`
		} `yaml:"cosign"`
	} `yaml:"import"`
}

type imageConfigSection struct {
	Ref   string `yaml:"ref"`
	Patch *bool  `yaml:"patch"`
}

type registryConfigSection struct {
	Name         string `yaml:"name"`
	URL          string `yaml:"url"`
	Insecure     bool   `yaml:"insecure"`
	PlainHTTP    bool   `yaml:"plainHTTP"`
	SourcePrefix bool   `yaml:"sourcePrefix"`
}

type ParserConfigSection struct {
	DisableImageDetection bool `yaml:"disableImageDetection"`
	UseCustomValues       bool `yaml:"useCustomValues"`
	FailOnMissingValues   bool `yaml:"failOnMissingValues"`
	FailOnMissingImages   bool `yaml:"failOnMissingImages"`
}

type MirrorConfigSection struct {
	Registry string `yaml:"registry"`
	Mirror   string `yaml:"mirror"`
}

func ConvertToHelmMirrors(configs []MirrorConfigSection) []helm.Mirror {
	var mirrors []helm.Mirror
	for _, config := range configs {
		mirrors = append(mirrors, helm.Mirror{
			Registry: config.Registry,
			Mirror:   config.Mirror,
		})
	}
	return mirrors
}

type config struct {
	Parser       ParserConfigSection     `yaml:"parser"`
	ImportConfig ImportConfigSection     `yaml:"import"`
	Images       []imageConfigSection    `yaml:"images"`
	Registries   []registryConfigSection `yaml:"registries"`
	Mirrors      []MirrorConfigSection   `yaml:"mirrors"`
}

// Reads flags from user and sets state accordingly
func LoadViperConfiguration() (*viper.Viper, error) {
	viper := viper.New()

	pflag.String("f", "unused", "path to configuration file")

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// Configure Viper configuration paths
	viper.SetConfigName("helmper") // name of config file (without extension)
	viper.SetConfigType("yaml")    // REQUIRED if the config file does not have the extension in the name

	if viper.GetString("f") == "unused" {
		viper.AddConfigPath("/etc/helmper/")         // path to look for the config file in
		viper.AddConfigPath("$HOME/.config/helmper") // call multiple times to add many search paths
		viper.AddConfigPath(".")                     // optionally look for config in the working directory
	} else {
		path := viper.GetString("f")
		viper.SetConfigFile(path)
	}

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		return nil, err
	}

	// set default values
	viper.SetDefault("all", false)
	viper.SetDefault("verbose", false)
	viper.SetDefault("update", false)
	viper.SetDefault("k8s_version", "1.31.1")

	// Unmarshal registries config section
	conf := config{}
	if err := viper.Unmarshal(&conf); err != nil {
		return nil, err
	}
	viper.Set("config", conf)
	viper.Set("parserConfig", conf.Parser)
	viper.Set("mirrorConfig", conf.Mirrors)

	// Unmarshal charts config section
	inputConf := helm.ChartCollection{}
	if err := viper.Unmarshal(&inputConf); err != nil {
		return nil, err
	}

	for _, c := range inputConf.Charts {
		rc, _ := helm.NewRegistryClient(c.PlainHTTP, false)
		if strings.HasPrefix(c.Repo.URL, "oci://") {
			rc = helm.NewOCIRegistryClient(rc, c.PlainHTTP)
		}

		c.RegistryClient = rc
		c.IndexFileLoader = &helm.FunctionLoader{
			LoadFunc: repo.LoadIndexFile,
		}

		if c.ValuesFilePath != "" && len(c.Values) > 0 {
			return nil, xerrors.Errorf("invalid chart configuration: cannot have both ValuesFilePath and Values defined at the same time")
		}

		if c.ValuesFilePath == "" && len(c.Values) > 0 {
			// write c.Values into a temp file and set c.ValuesFilePath to its path

			yamlBytes, err := yaml.Marshal(c.Values)
			if err != nil {
				return nil, xerrors.Errorf("failed to marshal values to YAML: %w", err)
			}

			tmpValuesFile, err := os.CreateTemp("", "chart_values_*.yaml")
			if err != nil {
				return nil, xerrors.Errorf("failed to create temp file: %w", err)
			}
			defer tmpValuesFile.Close()

			if _, err := tmpValuesFile.Write(yamlBytes); err != nil {
				return nil, xerrors.Errorf("failed to write YAML to temp file: %w", err)
			}

			c.ValuesFilePath = tmpValuesFile.Name()
		}

		if conf.Parser.FailOnMissingValues {
			if c.ValuesFilePath == "" {
				continue
			}
			if !file.Exists(c.ValuesFilePath) {
				return nil, xerrors.Errorf("values file %s does not exist", c.ValuesFilePath)
			}
		}
	}
	viper.Set("input", inputConf)

	importConf := ImportConfigSection{}
	if err := viper.Unmarshal(&importConf); err != nil {
		return nil, err
	}

	if importConf.Import.Cosign.Enabled && importConf.Import.Cosign.KeyRef == "" {
		s := `
import:
  cosign:
    enabled: true
    keyRef: ""     <---
`
		return nil, xerrors.Errorf("You have enabled cosign but did not specify any keyRef. Please specify a keyRef and try again..\nExample config:\n%s", s)
	}

	if importConf.Import.Cosign.Enabled && importConf.Import.Cosign.KeyRefPass == nil {
		v := os.Getenv("COSIGN_PASSWORD")
		slog.Info("KeyRefPass is nil, using value of COSIGN_PASSWORD environment variable")
		importConf.Import.Cosign.KeyRefPass = &v
	}

	if importConf.Import.Cosign.Enabled && importConf.Import.Cosign.PubKeyRef == nil {
		keyRef := importConf.Import.Cosign.KeyRef
		if strings.HasSuffix(keyRef, ".key") {
			keyRef = strings.Replace(keyRef, ".key", ".pub", 1)
		}
		importConf.Import.Cosign.PubKeyRef = to.Ptr(keyRef)
	}

	if importConf.Import.Copacetic.Enabled {

		if importConf.Import.Copacetic.Buildkitd.Addr == "" {
			// use local socket by default
			importConf.Import.Copacetic.Buildkitd.Addr = "unix:///run/buildkit/buildkitd.sock"
		}

		if importConf.Import.Copacetic.Trivy.Addr == "" {
			s := `
import:
  copacetic:
    enabled: true
    trivy:
      addr: http://0.0.0.0:8887  <---
`
			return nil, xerrors.Errorf("You have enabled copacetic patching but did not specify the path to the Trivy server. Please add the value and try again...\nExample config:\n%s", s)
		}
		viper.OnConfigChange(func(e fsnotify.Event) {
			slog.Info("Config file changed. It will not take effect before next run.", slog.String("config", e.Name))
		})
		viper.WatchConfig()

		if importConf.Import.Copacetic.Output.Reports.Folder == "" {
			s := `
copacetic:
  enabled: true
  output:
    reports:
      folder: /workspace/.out/reports  <---
`
			return nil, xerrors.Errorf("You have enabled copacetic patching but did not specify the path to the reports output folder'. Please add the value and try again\nExample:\n%s", s)
		}

		if importConf.Import.Copacetic.Output.Tars.Folder == "" {
			s := `
copacetic:
  enabled: true
  output:
    tars:
      folder: /workspace/.out/tars  <---
`
			return nil, xerrors.Errorf("You have enabled copacetic patching but did not specify the path to the tars output folder'. Please add the value and try again\nExample:\n%s", s)
		}

	}

	viper.Set("importConfig", importConf)

	rs := []*registry.Registry{}
	for _, r := range conf.Registries {
		rs = append(rs,
			to.Ptr(registry.Registry{
				Name:         r.Name,
				URL:          r.URL,
				PlainHTTP:    r.PlainHTTP,
				Insecure:     r.Insecure,
				PrefixSource: r.SourcePrefix,
			}))
	}
	state.SetValue(viper, "registries", rs)

	// TODO. Concert config.Images to Image{}
	is := []image.Image{}
	for _, i := range conf.Images {
		img, err := image.RefToImage(i.Ref)
		if err != nil {
			return viper, err
		}
		is = append(is, img)
	}
	state.SetValue(viper, "images", is)

	viper.OnConfigChange(func(e fsnotify.Event) {
		slog.Info("Config file changed. It will not take effect before next run.", slog.String("config", e.Name))
	})
	viper.WatchConfig()

	return viper, nil
}

// Viper module for Fx
var ViperModule = fx.Provide(
	LoadViperConfiguration,
)
