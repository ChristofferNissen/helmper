package bootstrap

import (
	"log"
	"os"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"helm.sh/helm/v3/pkg/cli"
)

// Add Helm repos to user's local helm configuration file, Optionupdate all existing repos and pulls charts
func SetupHelm(settings *cli.EnvSettings, charts *helm.ChartCollection, setters ...helm.Option) (*helm.ChartCollection, error) {

	// Default Options
	args := &helm.Options{
		Verbose:    false,
		Update:     false,
		K8SVersion: "1.27.16",
	}

	for _, setter := range setters {
		setter(args)
	}

	// Set up Helm action configuration
	if err := os.Setenv("HELM_EXPERIMENTAL_OCI", "1"); err != nil {
		log.Fatalf("Error setting OCI environment variable: %v", err)
	}

	return charts.SetupHelm(
		settings,
		helm.Update(args.Update),
		helm.Verbose(args.Verbose),
	)
}
