package cosign

import (
	"context"
	"fmt"

	"log/slog"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/report"
	"github.com/ChristofferNissen/helmper/pkg/util/bar"
	"github.com/ChristofferNissen/helmper/pkg/util/counter"
	"github.com/ChristofferNissen/helmper/pkg/util/terminal"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/verify"
)

type VerifyOption struct {
	Data           map[*registry.Registry]map[*image.Image]bool
	VerifyExisting bool

	KeyRef            string
	AllowInsecure     bool
	AllowHTTPRegistry bool

	Report *report.Table
}

// VerifyOption wraps the cosign CLIs native code
func (vo *VerifyOption) Run(ctx context.Context) (map[*registry.Registry]map[*image.Image]bool, error) {

	if vo.Report == nil {
		vo.Report = report.NewTable("Signature Overview For Images")
	}

	var sc counter.SafeCounter = counter.NewSafeCounter()

	header := table.Row{"#", "Image"}

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
		slog.Debug("No images or registries specified. Skipping verifying images...")
		return make(map[*registry.Registry]map[*image.Image]bool), nil
	}

	bar := bar.New("Verifying signatures...\r", size)

	o := &options.VerifyOptions{
		Key:         vo.KeyRef,
		CheckClaims: true,
		Output:      "",
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
		return make(map[*registry.Registry]map[*image.Image]bool), err
	}

	hashAlgorithm, err := o.SignatureDigest.HashAlgorithm()
	if err != nil {
		return make(map[*registry.Registry]map[*image.Image]bool), err
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

	keys := make([]string, 0)
	rows := make(map[string]*table.Row)

	m := make(map[*registry.Registry]map[*image.Image]bool, 0)
	for r, elem := range vo.Data {
		if elem == nil {
			elem = make(map[*image.Image]bool, 0)
		}

		// extend table for each registry
		rn := r.GetName()
		header = append(header, rn)

		for i, b := range elem {
			// add row to overview table
			ref := i.String()

			// Check for existing row for Chart Name
			row := rows[ref]
			if row == nil {
				row = to.Ptr(table.Row{sc.Value("index_import"), ref})
				rows[ref] = row
				keys = append(keys, ref)
			}

			if b || vo.VerifyExisting {
				name, err := i.ImageName()
				if err != nil {
					return nil, err
				}
				if r.PrefixSource {
					old := name
					name, err = image.UpdateNameWithPrefixSource(i)
					if err != nil {
						return nil, err
					}
					slog.Info("registry has PrefixSource enabled", slog.String("old", old), slog.String("new", name))
				}

				if !b {
					if i.Digest == "" {
						d, err := r.Fetch(ctx, name, i.Tag)
						if err != nil {
							return nil, err
						}
						i.Digest = d.Digest.String()
					}
				}

				out, err := terminal.CaptureOutput(func() error {
					url, _ := strings.CutPrefix(r.URL, "oci://")
					url = strings.Replace(url, "0.0.0.0", "localhost", 1)
					s := fmt.Sprintf("%s/%s@%s", url, name, i.Digest)
					return v.Exec(ctx, []string{s})
				})
				slog.Debug(out)

				if err != nil {
					switch {
					case isNoCertificateFoundOnSignatureErr(err):
						fallthrough
					case isNoMatchingSignatureErr(err):
						fallthrough
					case isImageWithoutSignatureErr(err):
						elem[i] = true
					default:
						return make(map[*registry.Registry]map[*image.Image]bool), err
					}
				}

				elem[i] = false
				*row = append(*row, terminal.StatusEmoji(!elem[i]))
				sc.Inc("index_import")
				_ = bar.Add(1)
			}

		}
		m[r] = elem
	}

	// Output table
	for _, k := range keys {
		valP := rows[k]
		if valP != nil {
			vo.Report.AddRow(*valP)
		}
	}
	vo.Report.AddHeader(header)

	_ = bar.Finish()

	return m, nil
}
