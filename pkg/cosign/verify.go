package cosign

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/verify"
)

type VerifyOption struct {
	Refs       []*registry.Image
	Registries []registry.Registry

	KeyRef            string
	AllowInsecure     bool
	AllowHTTPRegistry bool
}

// VerifyOption wraps the cosign CLIs native code
func (vo VerifyOption) Run() (map[registry.Registry]map[*registry.Image]bool, error) {

	// Return early: no images to sign, or no registries to upload signature to
	if !(len(vo.Refs) > 0) || !(len(vo.Registries) >= 0) {
		slog.Debug("No images or registries specified. Skipping signing images...")
		return make(map[registry.Registry]map[*registry.Image]bool), nil
	}

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
		return make(map[registry.Registry]map[*registry.Image]bool), err
	}

	hashAlgorithm, err := o.SignatureDigest.HashAlgorithm()
	if err != nil {
		return make(map[registry.Registry]map[*registry.Image]bool), err
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

	m := make(map[registry.Registry]map[*registry.Image]bool, 0)
	for _, r := range vo.Registries {
		elem := m[r]
		if elem == nil {
			elem = make(map[*registry.Image]bool, 0)
		}

		for _, i := range vo.Refs {
			name, _ := i.ImageName()
			s := fmt.Sprintf("%s/%s@%s", r.URL, name, i.Digest)
			err := v.Exec(ctx, []string{s})
			if err != nil {
				switch err.Error() {
				case "no signatures found":
					elem[i] = false
					continue
				default:
					return make(map[registry.Registry]map[*registry.Image]bool), err
				}
			}
			elem[i] = true
		}
		m[r] = elem
	}

	return m, nil
}
