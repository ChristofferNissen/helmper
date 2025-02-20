package bootstrap

import (
	"log/slog"
	"os"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"go.uber.org/fx"
	"helm.sh/helm/v3/pkg/cli"
)

// EnvironmentSetter is a function type for setting environment variables
type EnvironmentSetter func(key, value string) error

var setEnv EnvironmentSetter = os.Setenv

type ChartSetupper interface {
	SetupHelm(settings *cli.EnvSettings, setters ...helm.Option) (*helm.ChartCollection, error)
}

// Add Helm repos to user's local helm configuration file, Optionupdate all existing repos and pulls charts
func SetupHelm(settings *cli.EnvSettings, charts ChartSetupper, setters ...helm.Option) (*helm.ChartCollection, error) {
	// Default Options
	args := &helm.Options{
		Verbose:    false,
		Update:     false,
		K8SVersion: "1.31.1",
	}

	for _, setter := range setters {
		setter(args)
	}

	// Set up Helm action configuration
	if err := setEnv("HELM_EXPERIMENTAL_OCI", "1"); err != nil {
		slog.Error("Error setting OCI environment variable", slog.Any("error", err))
		os.Exit(1)
	}

	return charts.SetupHelm(
		settings,
		helm.Update(args.Update),
		helm.Verbose(args.Verbose),
	)
}

func ProvideHelmSettings() *cli.EnvSettings {
	return cli.New()
}

var HelmSettingsModule = fx.Provide(ProvideHelmSettings)
