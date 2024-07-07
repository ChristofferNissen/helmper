package helm

import (
	"bytes"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/exp/slog"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
)

// Contructs a Helm Chart Downloader Manager from Helm SDK
func getManager(out *bytes.Buffer, verbose bool, update bool) downloader.Manager {
	httpGetter := func(options ...getter.Option) (getter.Getter, error) {
		// Get retryable logic
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = 10
		retryClient.RetryWaitMin = time.Second * 1
		retryClient.RetryWaitMax = time.Second * 10
		transport := retryClient.HTTPClient.Transport.(*http.Transport)

		// Set options
		o1 := getter.WithTimeout(10 * time.Second)
		o2 := getter.WithTransport(transport)
		// if plainHTTP and insecure set
		o3 := getter.WithPlainHTTP(true)
		o4 := getter.WithInsecureSkipVerifyTLS(true)
		opts := append(options, []getter.Option{o1, o2, o3, o4}...)

		// return curried function
		return getter.NewHTTPGetter(opts...)
	}

	OCIGetter := func(options ...getter.Option) (getter.Getter, error) {
		o3 := getter.WithPlainHTTP(true)
		o4 := getter.WithInsecureSkipVerifyTLS(true)
		opts := append(options, []getter.Option{o3, o4}...)
		return getter.NewOCIGetter(opts...)
	}

	cl := cli.New()

	// TODO: Handle error
	rClient, _ := registry.NewRegistryClientWithTLS(out, "", "", "", false, cl.RegistryConfig, false)
	// if err != nil {

	// }
	return downloader.Manager{
		Out:              out,
		RegistryClient:   rClient,
		RepositoryConfig: cl.RepositoryConfig,
		RepositoryCache:  cl.RepositoryCache,
		Verify:           downloader.VerifyIfPossible,
		Debug:            verbose,
		SkipUpdate:       !update,
		Getters: []getter.Provider{
			{
				Schemes: []string{registry.OCIScheme},
				New:     OCIGetter,
			},
			{
				Schemes: []string{"http", "https"},
				New:     httpGetter,
			},
		},
	}
}

func updateRepository(path string, opts ...Option) error {

	// Default Options
	args := &Options{
		Verbose: false,
		Update:  false,
	}

	for _, opt := range opts {
		opt(args)
	}

	// Update Helm Repos
	var out bytes.Buffer
	ma := getManager(&out, args.Verbose, args.Update)
	if args.Verbose {
		slog.Info(out.String())
	}
	ma.ChartPath = path
	return ma.Update()
}

// update all repositories in local configuration file
func updateRepositories(verbose, update bool) (string, error) {

	// Update Helm Repos
	var out bytes.Buffer
	ma := getManager(&out, verbose, update)

	err := ma.UpdateRepositories()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}
