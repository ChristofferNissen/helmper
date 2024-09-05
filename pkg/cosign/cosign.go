package cosign

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/sign"

	_ "github.com/sigstore/sigstore/pkg/signature/kms/aws"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/azure"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/fake"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/gcp"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/hashivault"
)

type SignOption struct {
	Imgs       []*registry.Image
	Registries []registry.Registry

	KeyRef            string
	KeyRefPass        string
	AllowInsecure     bool
	AllowHTTPRegistry bool
}

// cosignAdapter wraps the cosign CLIs native code
func (so SignOption) Run() error {

	// Return early i no images to sign, or no registries to upload signature to
	if !(len(so.Imgs) > 0) || !(len(so.Registries) >= 0) {
		slog.Debug("No images or registries specified. Skipping signing images...")
		return nil
	}

	bar := progressbar.NewOptions(len(so.Imgs), progressbar.OptionSetWriter(ansi.NewAnsiStdout()), // "github.com/k0kubun/go-ansi"
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetDescription("Signing images...\r"),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	// Sign with cosign
	timeout := 2 * time.Minute
	ro := options.RootOptions{
		Timeout: timeout,
		Verbose: false,
	}

	signOpts := options.SignOptions{
		Key: so.KeyRef,

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

	for _, r := range so.Registries {
		refs := []string{}
		for _, i := range so.Imgs {
			name, _ := i.ImageName()
			ref := fmt.Sprintf("%s/%s@%s", r.URL, name, i.Digest)
			refs = append(refs, ref)
		}
		if err := sign.SignCmd(&ro, ko, signOpts, refs); err != nil {
			return err
		}
		_ = bar.Add(len(refs))
	}

	_ = bar.Finish()

	return nil
}
