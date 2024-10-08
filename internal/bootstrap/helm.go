package bootstrap

import (
	"github.com/ChristofferNissen/helmper/pkg/helm"
)

// Add Helm repos to user's local helm configuration file, Optionupdate all existing repos and pulls charts
func SetupHelm(charts *helm.ChartCollection, setters ...helm.Option) (helm.ChartCollection, error) {

	// Default Options
	args := &helm.Options{
		Verbose:    false,
		Update:     false,
		K8SVersion: "1.27.16",
	}

	for _, setter := range setters {
		setter(args)
	}

	return charts.SetupHelm(
		helm.Update(args.Update),
		helm.Verbose(args.Verbose),
	)
}
