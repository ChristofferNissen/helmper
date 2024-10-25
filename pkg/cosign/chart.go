package cosign

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/util/bar"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/sign"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"

	_ "github.com/sigstore/sigstore/pkg/signature/kms/aws"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/azure"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/fake"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/gcp"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/hashivault"
)

type SignChartOption struct {
	// ChartCollection *helm.ChartCollection
	// Registries      []registry.Registry
	Data map[*registry.Registry]map[*helm.Chart]bool

	KeyRef            string
	KeyRefPass        string
	AllowInsecure     bool
	AllowHTTPRegistry bool

	Settings *cli.EnvSettings
}

// cosignAdapter wraps the cosign CLIs native code
func (so SignChartOption) Run() error {

	size := func() int {
		size := 0
		for _, m := range so.Data {
			for _, b := range m {
				if b {
					size++
				}
			}
		}
		return size
	}()

	// Return early i no charts to sign, or no registries to upload signature to
	if !(size > 0) {
		slog.Debug("No charts or registries specified. Skipping signing charts...")
		return nil
	}

	if so.Settings == nil {
		so.Settings = cli.New()
	}

	bar := bar.New("Signing charts...\r", size)

	// Sign with cosign
	timeout := 2 * time.Minute
	ro := options.RootOptions{
		Timeout: timeout,
		Verbose: false,
	}

	signOpts := options.SignOptions{
		Key:              so.KeyRef,
		Upload:           true,
		TlogUpload:       false,
		SkipConfirmation: true,

		Registry: options.RegistryOptions{
			AllowInsecure:     so.AllowInsecure,
			AllowHTTPRegistry: so.AllowHTTPRegistry,

			RegistryClientOpts: []remote.Option{
				remote.WithAuthFromKeychain(authn.DefaultKeychain),
				remote.WithRetryBackoff(remote.Backoff{
					Duration: 1 * time.Second,
					Jitter:   1.0,
					Factor:   2.0,
					Steps:    5,
					Cap:      timeout,
				}),
			},
		},
	}

	oidcClientSecret, err := signOpts.OIDC.ClientSecret()
	if err != nil {
		return err
	}
	ko := options.KeyOpts{
		KeyRef:                         signOpts.Key,
		PassFunc:                       func(bool) ([]byte, error) { return []byte(so.KeyRefPass), nil },
		Sk:                             signOpts.SecurityKey.Use,
		Slot:                           signOpts.SecurityKey.Slot,
		FulcioURL:                      signOpts.Fulcio.URL,
		IDToken:                        signOpts.Fulcio.IdentityToken,
		FulcioAuthFlow:                 signOpts.Fulcio.AuthFlow,
		InsecureSkipFulcioVerify:       signOpts.Fulcio.InsecureSkipFulcioVerify,
		RekorURL:                       signOpts.Rekor.URL,
		OIDCIssuer:                     signOpts.OIDC.Issuer,
		OIDCClientID:                   signOpts.OIDC.ClientID,
		OIDCClientSecret:               oidcClientSecret,
		OIDCRedirectURL:                signOpts.OIDC.RedirectURL,
		OIDCDisableProviders:           signOpts.OIDC.DisableAmbientProviders,
		OIDCProvider:                   signOpts.OIDC.Provider,
		SkipConfirmation:               signOpts.SkipConfirmation,
		TSAClientCACert:                signOpts.TSAClientCACert,
		TSAClientCert:                  signOpts.TSAClientCert,
		TSAClientKey:                   signOpts.TSAClientKey,
		TSAServerName:                  signOpts.TSAServerName,
		TSAServerURL:                   signOpts.TSAServerURL,
		IssueCertificateForExistingKey: signOpts.IssueCertificate,
	}

	for r, m := range so.Data {
		refs := []string{}
		for c, b := range m {
			if !b {
				continue
			}

			name := fmt.Sprintf("%s/%s", chartutil.ChartsDir, c.Name)
			d, err := r.Fetch(context.TODO(), name, c.Version)
			if err != nil {
				return err
			}

			url, _ := strings.CutPrefix(r.URL, "oci://")
			url = strings.Replace(url, "0.0.0.0", "localhost", 1)
			ref := fmt.Sprintf("%s/%s/%s@%s", url, chartutil.ChartsDir, c.Name, d.Digest)
			refs = append(refs, ref)
		}

		// bar.ChangeMax(size + len(refs) - 1)
		if err := sign.SignCmd(&ro, ko, signOpts, refs); err != nil {
			return err
		}
		_ = bar.Add(len(refs))
	}

	_ = bar.Finish()

	return nil
}
