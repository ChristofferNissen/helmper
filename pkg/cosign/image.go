package cosign

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"helm.sh/helm/v3/pkg/cli"

	"github.com/ChristofferNissen/helmper/pkg/image"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/report"
	"github.com/ChristofferNissen/helmper/pkg/util/bar"
	"github.com/ChristofferNissen/helmper/pkg/util/terminal"
)

type SignImageOption struct {
	CommonOption
	Data map[*registry.Registry]map[*image.Image]bool
}

type VerifyImageOption struct {
	CommonOption
	Data           map[*registry.Registry]map[*image.Image]bool
	VerifyExisting bool

	Report *report.Table
}

// Sign images using the cosign CLI
func (so SignImageOption) Run(ctx context.Context) error {
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

	if !(size > 0) {
		slog.Debug("No images or registries specified. Skipping signing images...")
		return nil
	}

	if so.Settings == nil {
		so.Settings = cli.New()
	}

	bar := bar.New("Signing images...\r", size)

	for r, m := range so.Data {
		refs := []string{}
		for i, b := range m {
			if b {
				name, _ := i.ImageName()
				if i.Digest == "" {
					d, err := r.Fetch(ctx, name, i.Tag)
					if err != nil {
						return err
					}
					i.Digest = d.Digest.String()
				}
				if r.PrefixSource {
					old := name
					name, _ = image.UpdateNameWithPrefixSource(i)
					slog.Info("registry has PrefixSource enabled", slog.String("old", old), slog.String("new", name))
				}
				url, _ := strings.CutPrefix(r.URL, "oci://")
				url = strings.Replace(url, "0.0.0.0", "localhost", 1)
				ref := fmt.Sprintf("%s/%s@%s", url, name, i.Digest)
				refs = append(refs, ref)
			}
		}

		for _, ref := range refs {
			cmd := exec.Command("cosign", "sign", "-key", so.KeyRef, ref)
			cmd.Env = append(cmd.Env, "COSIGN_PASSWORD="+so.KeyRefPass)
			output, err := cmd.CombinedOutput()
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to sign %s: %v, output: %s", ref, err, string(output)))
				return fmt.Errorf("failed to sign %s: %v, output: %s", ref, err, string(output))
			}
			slog.Info(fmt.Sprintf("Successfully signed %s", ref))
		}

		_ = bar.Add(len(refs))
	}

	_ = bar.Finish()

	return nil
}

// Verify images using the cosign CLI
func (vo VerifyImageOption) Run(ctx context.Context) (map[*registry.Registry]map[*image.Image]bool, error) {
	if vo.Report == nil {
		vo.Report = report.NewTable("Signature Overview For Images")
	}

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

	if !(size > 0) {
		slog.Debug("No images or registries specified. Skipping verifying images...")
		return make(map[*registry.Registry]map[*image.Image]bool), nil
	}

	if vo.Settings == nil {
		vo.Settings = cli.New()
	}

	bar := bar.New("Verifying images...\r", size)

	m := make(map[*registry.Registry]map[*image.Image]bool, 0)
	keys := make([]string, 0)
	rows := make(map[string]*table.Row)
	header := table.Row{"#", "Image"}

	for r, elem := range vo.Data {
		if elem == nil {
			elem = make(map[*image.Image]bool, 0)
		}

		rn := r.GetName()
		header = append(header, rn)

		for i, b := range elem {
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
					cmd := exec.Command("cosign", "verify", "-key", vo.KeyRef, s)
					cmd.Env = append(cmd.Env, "COSIGN_PASSWORD="+vo.KeyRefPass)
					output, err := cmd.CombinedOutput()
					slog.Debug(string(output))
					return err
				})
				if err != nil {
					switch {
					case isNoCertificateFoundOnSignatureErr(out):
						fallthrough
					case isNoMatchingSignatureErr(out):
						fallthrough
					case isImageWithoutSignatureErr(out):
						elem[i] = true
					default:
						slog.Error(fmt.Sprintf("verification error: %v, output: %s", err, out))
						return make(map[*registry.Registry]map[*image.Image]bool), err
					}
				} else {
					elem[i] = false
				}

				ref := i.String()
				row := rows[ref]
				if row == nil {
					row = &table.Row{len(keys) + 1, ref}
					rows[ref] = row
					keys = append(keys, ref)
				}
				*row = append(*row, terminal.StatusEmoji(!elem[i]))
				_ = bar.Add(1)
			}
		}
		m[r] = elem
	}

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
