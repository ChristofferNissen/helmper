package trivy

import (
	"context"
	"fmt"
	"log/slog"

	tcache "github.com/aquasecurity/trivy/pkg/cache"
	"github.com/aquasecurity/trivy/pkg/fanal/analyzer"
	"github.com/aquasecurity/trivy/pkg/fanal/artifact"
	image2 "github.com/aquasecurity/trivy/pkg/fanal/artifact/image"
	"github.com/aquasecurity/trivy/pkg/fanal/image"
	ftypes "github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/result"
	"github.com/aquasecurity/trivy/pkg/rpc/client"
	"github.com/aquasecurity/trivy/pkg/scanner"
	"github.com/aquasecurity/trivy/pkg/types"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/samber/lo"

	dbTypes "github.com/aquasecurity/trivy-db/pkg/types"

	_ "modernc.org/sqlite" // sqlite driver for RPM DB and Java DB
)

type ScanOption struct {
	DockerHost    string
	TrivyServer   string
	Insecure      bool
	IgnoreUnfixed bool
	Architecture  *string
	Local         bool
}

func (opts ScanOption) Scan(reference string) (*types.Report, error) {

	platform := ftypes.Platform{}
	if opts.Architecture != nil {
		p, _ := v1.ParsePlatform(*opts.Architecture)
		platform = ftypes.Platform{
			Platform: p,
		}
	}

	clientScanner := client.NewScanner(client.ScannerOption{
		RemoteURL: opts.TrivyServer,
		Insecure:  opts.Insecure,
	}, []client.Option(nil)...)

	var typesImage ftypes.Image
	if opts.Local {
		img, err := image.NewArchiveImage(reference)
		if err != nil {
			slog.Error("NewArchiveImage failed: %v", err)
			return nil, err
		}
		typesImage = img
		// configFile, err := img.ConfigFile()
		// if err != nil {
		// 	slog.Error("ConfigFile failed: %v", err)
		// 	return types.Report{}, err
		// }
		// slog.Info(fmt.Sprintf("Image reference: %s", configFile.Config.Labels["org.opencontainers.image.ref.name"]))

	} else {
		ti, cleanup, err := image.NewContainerImage(context.TODO(), reference, ftypes.ImageOptions{
			RegistryOptions: ftypes.RegistryOptions{
				Insecure: opts.Insecure,
				Platform: platform,
			},
			DockerOptions: ftypes.DockerOptions{
				Host: opts.DockerHost,
			},
			ImageSources: []ftypes.ImageSource{ftypes.RemoteImageSource},
		})
		if err != nil {
			slog.Error("NewContainerImage failed: %v", err)
			return nil, err
		}
		typesImage = ti
		defer cleanup()
	}

	cache := tcache.NewRemoteCache(
		tcache.RemoteOptions{
			ServerAddr: opts.TrivyServer,
			Insecure:   opts.Insecure,
		})
	// cache := tcache.NopCache(remoteCache)

	artifactArtifact, err := image2.NewArtifact(typesImage, cache, artifact.Option{
		DisabledAnalyzers: []analyzer.Type{
			analyzer.TypeJar,
			analyzer.TypePom,
			analyzer.TypeGradleLock,
			analyzer.TypeSbtLock,
		},
		DisabledHandlers: nil,
		FilePatterns:     nil,
		NoProgress:       false,
		Insecure:         opts.Insecure,
		SBOMSources:      nil,
		RekorURL:         "https://rekor.sigstore.dev",
		ImageOption: ftypes.ImageOptions{
			RegistryOptions: ftypes.RegistryOptions{
				Insecure: opts.Insecure,
				Platform: platform,
			},
			DockerOptions: ftypes.DockerOptions{
				Host: opts.DockerHost,
			},
			ImageSources: []ftypes.ImageSource{ftypes.RemoteImageSource},
		},
	})
	if err != nil {
		slog.Error("NewArtifact failed: %v", err)
		return nil, err
	}

	scannerScanner := scanner.NewScanner(clientScanner, artifactArtifact)
	report, err := scannerScanner.ScanArtifact(context.TODO(), types.ScanOptions{
		PkgTypes:            []string{types.PkgTypeOS},
		Scanners:            types.AllScanners,
		ImageConfigScanners: types.AllImageConfigScanners,
		ScanRemovedPackages: false,
		FilePatterns:        nil,
		IncludeDevDeps:      false,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("ScanArtifact failed: %v", err), slog.Any("report", report))
		return nil, err
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

	return &report, nil

}
