package trivy

import (
	"context"
	"encoding/json"
	"log/slog"
	"os/exec"

	"github.com/aquasecurity/trivy/pkg/result"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/samber/lo"

	dbTypes "github.com/aquasecurity/trivy-db/pkg/types"
)

type ScanOption struct {
	DockerHost    string
	TrivyServer   string
	Insecure      bool
	IgnoreUnfixed bool
	Architecture  *string
}

func (opts ScanOption) Scan(reference string) (types.Report, error) {
	args := []string{"image", "--format", "json"}

	if opts.Architecture != nil {
		args = append(args, "--platform", *opts.Architecture)
	}

	if opts.TrivyServer != "" {
		args = append(args, "--server", opts.TrivyServer)
	}

	if opts.Insecure {
		args = append(args, "--insecure")
	}

	if opts.IgnoreUnfixed {
		args = append(args, "--ignore-unfixed")
	}

	args = append(args, reference)

	cmd := exec.Command("trivy", args...)
	output, err := cmd.Output()
	if err != nil {
		slog.Error("Trivy CLI execution failed", slog.Any("error", err))
		return types.Report{}, err
	}

	var report types.Report
	err = json.Unmarshal(output, &report)
	if err != nil {
		slog.Error("Failed to parse Trivy CLI output", slog.Any("error", err))
		return types.Report{}, err
	}

	if opts.IgnoreUnfixed {
		ignoreStatuses := lo.FilterMap(
			dbTypes.Statuses,
			func(s string, _ int) (dbTypes.Status, bool) {
				fixed := dbTypes.StatusFixed
				if s == fixed.String() {
					return 0, false
				}
				return dbTypes.NewStatus(s), true
			},
		)

		result.Filter(context.TODO(), report, result.FilterOptions{
			Severities: []dbTypes.Severity{
				dbTypes.SeverityCritical,
				dbTypes.SeverityHigh,
			},
			IgnoreStatuses: ignoreStatuses,
		})
	}

	return report, nil
}
