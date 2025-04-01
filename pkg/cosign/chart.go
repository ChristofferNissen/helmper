package cosign

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"

	"github.com/ChristofferNissen/helmper/pkg/helm"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/report"
	"github.com/ChristofferNissen/helmper/pkg/util/bar"
	"github.com/ChristofferNissen/helmper/pkg/util/terminal"
)

type CommonOption struct {
	KeyRef            string
	KeyRefPass        string
	AllowInsecure     bool
	AllowHTTPRegistry bool

	Settings *cli.EnvSettings
}

type SignChartOption struct {
	CommonOption
	Data map[*registry.Registry]map[*helm.Chart]bool
}

type VerifyChartOption struct {
	CommonOption
	Data           map[*registry.Registry]map[*helm.Chart]bool
	VerifyExisting bool

	Report *report.Table
}

// Sign charts using the cosign CLI
func (so *SignChartOption) Run() error {
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

	if !(size > 0) {
		slog.Debug("No charts or registries specified. Skipping signing charts...")
		return nil
	}

	if so.Settings == nil {
		so.Settings = cli.New()
	}

	bar := bar.New("Signing charts...\r", size)

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

			digest := d.Digest.String()
			url, _ := strings.CutPrefix(r.URL, "oci://")
			url = strings.Replace(url, "0.0.0.0", "localhost", 1)
			ref := fmt.Sprintf("%s/%s/%s@%s", url, chartutil.ChartsDir, c.Name, digest)
			refs = append(refs, ref)
		}

		for _, ref := range refs {
			cmd := exec.Command("cosign", "sign", "--tlog-upload=false", "--allow-http-registry=true", "--allow-insecure-registry=true", "--key", so.KeyRef, ref)
			// slog.Info(fmt.Sprintf("cosign sign --tlog-upload false --key %s %s", so.KeyRef, ref))
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

// Verify charts using the cosign CLI
func (vo *VerifyChartOption) Run(ctx context.Context) (map[*registry.Registry]map[*helm.Chart]bool, error) {
	if vo.Report == nil {
		vo.Report = report.NewTable("Signature Overview For Charts")
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
		slog.Debug("No charts or registries specified. Skipping verifying charts...")
		return make(map[*registry.Registry]map[*helm.Chart]bool), nil
	}

	if vo.Settings == nil {
		vo.Settings = cli.New()
	}

	bar := bar.New("Verifying charts...\r", size)

	m := make(map[*registry.Registry]map[*helm.Chart]bool, 0)
	keys := make([]string, 0)
	rows := make(map[string]*table.Row)
	header := table.Row{"#", "Chart"}

	for r, elem := range vo.Data {
		if elem == nil {
			elem = make(map[*helm.Chart]bool, 0)
		}

		rn := r.GetName()
		header = append(header, rn)

		for c, b := range elem {
			if b || vo.VerifyExisting {
				name := fmt.Sprintf("%s/%s", chartutil.ChartsDir, c.Name)
				d, err := r.Fetch(ctx, name, c.Version)
				if err != nil {
					return nil, err
				}

				digest := d.Digest.String()
				out, err := terminal.CaptureOutput(func() error {
					url, _ := strings.CutPrefix(r.URL, "oci://")
					url = strings.Replace(url, "0.0.0.0", "localhost", 1)
					s := fmt.Sprintf("%s/%s/%s@%s", url, chartutil.ChartsDir, c.Name, digest)
					cmd := exec.Command("cosign", "verify", "--key", vo.KeyRef, s)
					slog.Info(fmt.Sprintf("cosign verify --key %s %s", vo.KeyRef, s))
					cmd.Env = append(cmd.Env, "COSIGN_PASSWORD="+vo.KeyRefPass)
					output, err := cmd.CombinedOutput()
					slog.Debug(string(output))

					if err != nil {
						switch {
						case isNoCertificateFoundOnSignatureErr(string(output)):
							fallthrough
						case isNoMatchingSignatureErr(string(output)):
							fallthrough
						case isImageWithoutSignatureErr(string(output)):
							elem[c] = true
						default:
							return err
						}
					} else {
						elem[c] = false
					}
					return nil
				})
				if err != nil {
					slog.Error(fmt.Sprintf("verification error: %v, output: %s", err, out))
					return make(map[*registry.Registry]map[*helm.Chart]bool), err
				}

				ref := c.Name
				row := rows[ref]
				if row == nil {
					row = &table.Row{len(keys) + 1, ref}
					rows[ref] = row
					keys = append(keys, ref)
				}
				*row = append(*row, terminal.StatusEmoji(!elem[c]))
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
