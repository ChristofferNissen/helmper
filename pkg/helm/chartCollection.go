package helm

import (
	"log"
	"log/slog"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/ChristofferNissen/helmper/pkg/util/terminal"
	"helm.sh/helm/v3/pkg/cli"
)

func (collection ChartCollection) pull(settings *cli.EnvSettings) error {
	for _, chart := range collection.Charts {
		if _, err := chart.Pull(settings); err != nil {
			return err
		}
	}
	return nil
}

func (collection ChartCollection) addToHelmRepositoryConfig(settings *cli.EnvSettings) error {
	for _, c := range collection.Charts {
		if strings.HasPrefix(c.Repo.URL, "oci://") {
			continue
		}
		_, err := c.addToHelmRepositoryFile(settings)
		if err != nil {
			return err
		}

	}
	return nil
}

// configures helm and pulls charts to local fs
func (collection ChartCollection) SetupHelm(settings *cli.EnvSettings, setters ...Option) (*ChartCollection, error) {

	// Default Options
	args := &Options{
		Verbose: false,
		Update:  false,
	}

	for _, setter := range setters {
		setter(args)
	}

	// Add Helm Repos
	err := collection.addToHelmRepositoryConfig(settings)
	if err != nil {
		return nil, err
	}
	if args.Verbose {
		log.Printf("Added Helm repositories to config '%s' %s\n", settings.RepositoryConfig, terminal.GetCheckMarkEmoji())
	}

	// Update Helm Repos
	output, err := updateRepositories(settings, args.Verbose, args.Update)
	if err != nil {
		return nil, err
	}
	// Log results
	if args.Verbose {
		log.Printf("Updated all Helm repositories %s\n%s", terminal.GetCheckMarkEmoji(), output)
	} else {
		log.Printf("Updated all Helm repositories %s\n", terminal.GetCheckMarkEmoji())
	}

	// Expand collection if semantic version range
	res := []*Chart{}
	for _, c := range collection.Charts {
		vs, err := c.ResolveVersions(settings)
		if err != nil {
			// resolve Glob version
			v, err := c.ResolveVersion(settings)
			if err != nil {
				slog.Info("version is not semver. skipping this version", slog.String("name", c.Name), slog.String("version", c.Version))
				continue
			}
			c.Version = v
			res = append(res, c)
		}

		for _, v := range vs {
			c := c
			c.Version = v
			res = append(res, c)
		}
	}
	collection.Charts = res

	// Pull Helm Charts
	err = collection.pull(settings)
	if err != nil {
		return nil, err
	}
	if args.Verbose {
		log.Println("Pulled Helm Charts")
	}

	return to.Ptr(collection), nil
}
