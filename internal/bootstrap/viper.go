package bootstrap

import (
	"log/slog"
	"os"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/util/state"
	"github.com/distribution/reference"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"
)

type ImportConfigSection struct {
	Import struct {
		Enabled      bool    `yaml:"enabled"`
		Architecture *string `yaml:"architecture"`
		Copacetic    struct {
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
			KeyRef            string  `yaml:"keyRef"`
			KeyRefPass        *string `yaml:"keyRefPass"`
			AllowHTTPRegistry bool    `yaml:"allowHTTPRegistry"`
			AllowInsecure     bool    `yaml:"allowInsecure"`
		} `yaml:"cosign"`
	} `yaml:"import"`
}

type imageConfigSection struct {
	Ref string `yaml:"ref"`
}

type registryConfigSection struct {
	Name      string `yaml:"name"`
	URL       string `yaml:"url"`
	Insecure  bool   `yaml:"insecure"`
	PlainHTTP bool   `yaml:"plainHTTP"`
}

type config struct {
	ImportConfig ImportConfigSection     `yaml:"import"`
	Images       []imageConfigSection    `yaml:"images"`
	Registries   []registryConfigSection `yaml:"registries"`
}

// Reads flags from user and sets state accordingly
func LoadViperConfiguration(_ []string) (*viper.Viper, error) {
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

	// Unmarshal charts config section
	inputConf := helm.ChartCollection{}
	if err := viper.Unmarshal(&inputConf); err != nil {
		return nil, err
	}
	viper.Set("input", inputConf)

	// Unmarshal registries config section
	conf := config{}
	if err := viper.Unmarshal(&conf); err != nil {
		return nil, err
	}
	viper.Set("config", conf)

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

	rs := []registry.Registry{}
	for _, r := range conf.Registries {
		rs = append(rs,
			registry.Registry{
				Name:      r.Name,
				URL:       r.URL,
				PlainHTTP: r.PlainHTTP,
			})
	}
	state.SetValue(viper, "registries", rs)

	// TODO. Concert config.Images to Image{}
	is := []registry.Image{}
	for _, i := range conf.Images {
		ref, err := reference.ParseAnyReference(i.Ref)
		if err != nil {
			return viper, err
		}

		img := registry.Image{}
		switch r := ref.(type) {
		case reference.Canonical:
			d := reference.Domain(r)
			p := reference.Path(r)

			img.Registry = d
			img.Repository = p
			img.Digest = r.Digest().String()
			img.UseDigest = true

			if t, ok := r.(reference.Tagged); ok {
				img.Tag = t.Tag()
			}

			is = append(is, img)

		case reference.NamedTagged:
			d := reference.Domain(r)
			p := reference.Path(r)

			img.Registry = d
			img.Repository = p
			img.Tag = r.Tag()
			img.UseDigest = false

			is = append(is, img)

		}

	}
	state.SetValue(viper, "images", is)

	viper.OnConfigChange(func(e fsnotify.Event) {
		slog.Info("Config file changed. It will not take effect before next run.", slog.String("config", e.Name))
	})
	viper.WatchConfig()

	return viper, nil
}
