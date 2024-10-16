package cosign

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/verify"
)

type VerifyChartOption struct {
	Data           map[*registry.Registry]map[*helm.Chart]bool
	VerifyExisting bool

	KeyRef            string
	AllowInsecure     bool
	AllowHTTPRegistry bool
}

// VerifyOption wraps the cosign CLIs native code
func (vo VerifyChartOption) Run() (map[*registry.Registry]map[*helm.Chart]bool, error) {

	size := func() int {
		size := 0
		for _, m := range vo.Data {
			for _, b := range m {
				if b || vo.VerifyExisting {
					size++
				}
			}
		}
		return size
	}()

	// Return early: no images to sign, or no registries to upload signature to
	if !(size > 0) {
		slog.Debug("No charts or registries specified. Skipping verifying charts...")
		return make(map[*registry.Registry]map[*helm.Chart]bool), nil
	}

	bar := progressbar.NewOptions(size, progressbar.OptionSetWriter(ansi.NewAnsiStdout()), // "github.com/k0kubun/go-ansi"
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetDescription("Verifying signatures...\r"),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	o := &options.VerifyOptions{
		Key: vo.KeyRef,

		CheckClaims: true,

		Output: "json",

		CommonVerifyOptions: options.CommonVerifyOptions{
			IgnoreTlog:            true,
			PrivateInfrastructure: true,
			ExperimentalOCI11:     true,
		},

		Registry: options.RegistryOptions{
			AllowInsecure:     vo.AllowInsecure,
			AllowHTTPRegistry: vo.AllowHTTPRegistry,

			RegistryClientOpts: []remote.Option{
				remote.WithAuthFromKeychain(authn.DefaultKeychain),
				remote.WithRetryBackoff(remote.Backoff{
					Duration: 1 * time.Second,
					Jitter:   1.0,
					Factor:   2.0,
					Steps:    5,
					Cap:      1 * time.Minute,
				}),
			},
		},
	}

	annotations, err := o.AnnotationsMap()
	if err != nil {
		return make(map[*registry.Registry]map[*helm.Chart]bool), err
	}

	hashAlgorithm, err := o.SignatureDigest.HashAlgorithm()
	if err != nil {
		return make(map[*registry.Registry]map[*helm.Chart]bool), err
	}

	v := &verify.VerifyCommand{
		RegistryOptions:              o.Registry,
		CertVerifyOptions:            o.CertVerify,
		CheckClaims:                  o.CheckClaims,
		KeyRef:                       o.Key,
		CertRef:                      o.CertVerify.Cert,
		CertChain:                    o.CertVerify.CertChain,
		CAIntermediates:              o.CertVerify.CAIntermediates,
		CARoots:                      o.CertVerify.CARoots,
		CertGithubWorkflowTrigger:    o.CertVerify.CertGithubWorkflowTrigger,
		CertGithubWorkflowSha:        o.CertVerify.CertGithubWorkflowSha,
		CertGithubWorkflowName:       o.CertVerify.CertGithubWorkflowName,
		CertGithubWorkflowRepository: o.CertVerify.CertGithubWorkflowRepository,
		CertGithubWorkflowRef:        o.CertVerify.CertGithubWorkflowRef,
		IgnoreSCT:                    o.CertVerify.IgnoreSCT,
		SCTRef:                       o.CertVerify.SCT,
		Sk:                           o.SecurityKey.Use,
		Slot:                         o.SecurityKey.Slot,
		Output:                       o.Output,
		RekorURL:                     o.Rekor.URL,
		Attachment:                   o.Attachment,
		Annotations:                  annotations,
		HashAlgorithm:                hashAlgorithm,
		SignatureRef:                 o.SignatureRef,
		PayloadRef:                   o.PayloadRef,
		LocalImage:                   o.LocalImage,
		Offline:                      o.CommonVerifyOptions.Offline,
		TSACertChainPath:             o.CommonVerifyOptions.TSACertChainPath,
		IgnoreTlog:                   o.CommonVerifyOptions.IgnoreTlog,
		MaxWorkers:                   o.CommonVerifyOptions.MaxWorkers,
		ExperimentalOCI11:            o.CommonVerifyOptions.ExperimentalOCI11,
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	m := make(map[*registry.Registry]map[*helm.Chart]bool, 0)
	for r, elem := range vo.Data {
		if elem == nil {
			elem = make(map[*helm.Chart]bool, 0)
		}

		for c, b := range elem {
			if b || vo.VerifyExisting {
				name := fmt.Sprintf("charts/%s", c.Name)
				d, err := r.Fetch(ctx, name, c.Version)
				if err != nil {
					return make(map[*registry.Registry]map[*helm.Chart]bool), err
				}
				s := fmt.Sprintf("%s/charts/%s@%s", r.URL, c.Name, d.Digest)

				err = v.Exec(ctx, []string{s})
				if err != nil {
					switch err.Error() {
					case "no signatures found":
						elem[c] = true
						_ = bar.Add(1)
						continue
					default:
						return make(map[*registry.Registry]map[*helm.Chart]bool), err
					}
				}
				elem[c] = false
				_ = bar.Add(1)
			}
		}
		m[r] = elem
	}

	_ = bar.Finish()

	return m, nil
}
