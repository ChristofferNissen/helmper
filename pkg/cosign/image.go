package cosign

import (
	"context"
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
	Data map[*registry.Registry]map[*registry.Image]bool

	KeyRef            string
	KeyRefPass        string
	AllowInsecure     bool
	AllowHTTPRegistry bool
}

// SignOption wraps the cosign CLIs native code
func (so SignOption) Run() error {

	ctx := context.TODO()

	// count number of images
	size := func() int {
		i := 0
		for _, m := range so.Data {
			for _, b := range m {
				if b {
					i++
				}
			}
		}
		return i
	}()

	// Return early i no images to sign, or no registries to upload signature to
	if !(size > 0) {
		slog.Debug("No images or registries specified. Skipping signing images...")
		return nil
	}

	bar := progressbar.NewOptions(size, progressbar.OptionSetWriter(ansi.NewAnsiStdout()), // "github.com/k0kubun/go-ansi"
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

	for r, m := range so.Data {
		refs := []string{}
		for i, b := range m {
			if b {
				// if i.Registry == "docker.io" {
				// 	continue
				// }
				name, _ := i.ImageName()
				if i.Digest == "" {
					d, err := r.Fetch(ctx, name, i.Tag)
					if err != nil {
						return err
					}
					i.Digest = d.Digest.String()
				}
				ref := fmt.Sprintf("%s/%s@%s", r.URL, name, i.Digest)
				refs = append(refs, ref)
			}
		}
		if err := sign.SignCmd(&ro, ko, signOpts, refs); err != nil {
			return err
		}
		_ = bar.Add(len(refs))
	}

	_ = bar.Finish()

	return nil
}
