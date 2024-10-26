package flow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChristofferNissen/helmper/pkg/copa"
	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/pkg/trivy"
	myBar "github.com/ChristofferNissen/helmper/pkg/util/bar"
)

type SpsOption struct {
	Data          map[*registry.Registry]map[*registry.Image]bool
	All           bool
	Architecture  *string
	ReportsFolder string
	ReportsClean  bool
	TarsFolder    string
	TarsClean     bool
	ScanOption    trivy.ScanOption
	PatchOption   copa.PatchOption
}

func (o SpsOption) Run(ctx context.Context) error {
	// Count of images to import across registries
	lenImages := func() int {
		c := 0
		seen := make([]registry.Image, 0)
		for _, m := range o.Data {
			for i, b := range m {
				if b {
					if i.In(seen) {
						continue
					}
					seen = append(seen, *i)
					c++
				}
			}
		}
		return c
	}()

	if !(lenImages > 0) {
		return nil
	}

	// Trivy scan
	bar := myBar.New("Scanning images before patching...\r", lenImages)
	prescan := func() (map[*registry.Registry]map[*registry.Image]bool, map[*registry.Registry]map[*registry.Image]bool, error) {
		imgs := make([]*registry.Image, 0)
		for _, m := range o.Data {
			for i, b := range m {
				if b {
					if i.InP(imgs) {
						continue
					}
					imgs = append(imgs, i)
				}
			}
		}

		patch := make([]*registry.Image, 0)
		push := make([]*registry.Image, 0)
		for _, i := range imgs {
			if i.Patch != nil {
				if !*i.Patch {
					ref, _ := i.String()
					slog.Debug("image should not be patched", slog.String("image", ref))
					// pushRegistry[i] = true
					push = append(push, i)
					continue
				}
			}

			ref, err := i.String()
			if err != nil {
				return nil, nil, err
			}
			r, err := o.ScanOption.Scan(ref)
			if err != nil {
				return nil, nil, err
			}

			switch copa.SupportedOS(r.Metadata.OS) {
			case true:
				// filter images with no os-pkgs as copa has nothing to do
				switch trivy.ContainsOsPkgs(r.Results) {
				case true:
					slog.Debug("Image does contain os-pkgs vulnerabilities",
						slog.String("image", ref))
					patch = append(patch, i)
				case false:
					slog.Warn("Image does not contain os-pkgs. The image will not be patched.",
						slog.String("image", ref),
					)
					push = append(push, i)
				}
			case false:
				slog.Warn("Image contains an unsupported OS. The image will not be patched.",
					slog.String("image", ref),
				)
				push = append(push, i)
			}

			// Write report to filesystem
			name, _ := i.ImageName()
			fileName := fmt.Sprintf("%s:%s.json", name, i.Tag)
			fileName = filepath.Join(o.ReportsFolder, "prescan-"+strings.ReplaceAll(fileName, "/", "-"))
			b, err := json.MarshalIndent(r, "", "  ")
			if err != nil {
				return nil, nil, err
			}
			if err := os.WriteFile(fileName, b, os.ModePerm); err != nil {
				return nil, nil, err
			}
		}

		// filter images
		patchM := make(map[*registry.Registry]map[*registry.Image]bool, 0)
		pushM := make(map[*registry.Registry]map[*registry.Image]bool, 0)
		for r, elem := range o.Data {
			patchRegistry := patchM[r]
			if patchRegistry == nil {
				patchRegistry = make(map[*registry.Image]bool, 0)
				patchM[r] = patchRegistry
			}
			pushRegistry := pushM[r]
			if pushRegistry == nil {
				pushRegistry = make(map[*registry.Image]bool, 0)
				pushM[r] = pushRegistry
			}

			for i, b := range elem {
				if b {
					switch {
					case i.InP(patch):
						patchRegistry[i] = true
					case i.InP(push):
						pushRegistry[i] = true
					}

					_ = bar.Add(1)
				}
			}

			patchM[r] = patchRegistry
			pushM[r] = pushRegistry
		}

		return patchM, pushM, nil
	}
	patch, push, err := prescan()
	if err != nil {
		return err
	}
	_ = bar.Finish()

	// Determine fully qualified output path for images
	reportFilePaths := make(map[*registry.Image]string)
	reportPostFilePaths := make(map[*registry.Image]string)
	outFilePaths := make(map[*registry.Image]string)
	for _, elem := range o.Data {
		for i, b := range elem {
			if b {
				name, _ := i.ImageName()
				fileName := fmt.Sprintf("prescan-%s:%s.json", name, i.Tag)
				reportFilePaths[i] = filepath.Join(
					o.ReportsFolder,
					strings.ReplaceAll(fileName, "/", "-"),
				)
				fileName = fmt.Sprintf("postscan-%s:%s.json", name, i.Tag)
				reportPostFilePaths[i] = filepath.Join(
					o.ReportsFolder,
					strings.ReplaceAll(fileName, "/", "-"),
				)
				out := fmt.Sprintf("%s:%s.tar", name, i.Tag)
				outFilePaths[i] = filepath.Join(
					o.TarsFolder,
					strings.ReplaceAll(out, "/", "-"),
				)
			}
		}
	}
	// Clean up files
	defer func() {
		if o.ReportsClean {
			for _, v := range reportFilePaths {
				_ = os.RemoveAll(v)
			}
			for _, v := range reportPostFilePaths {
				_ = os.RemoveAll(v)
			}
		}
		if o.TarsClean {
			for _, v := range outFilePaths {
				_ = os.RemoveAll(v)
			}
		}
	}()

	// Import images without os-pkgs vulnerabilities
	io := registry.ImportOption{
		Data:         push,
		All:          o.All,
		Architecture: o.Architecture,
	}
	err = io.Run(context.WithoutCancel(ctx))
	if err != nil {
		return err
	}

	// Patch image and save to tar
	o.PatchOption.Data = patch
	err = o.PatchOption.Run(context.WithoutCancel(ctx), reportFilePaths, outFilePaths)
	if err != nil {
		return err
	}

	bar = myBar.New("Scanning images after patching...\r", lenImages)
	err = func(out string, prefix string) error {
		for _, m := range o.Data {
			for i, b := range m {
				if b {

					ref, err := i.String()
					if err != nil {
						return err
					}

					slog.Default().With(slog.String("image", ref))

					r, err := o.ScanOption.Scan(ref)
					if err != nil {
						return err
					}

					// Write report to filesystem
					name, _ := i.ImageName()
					fileName := fmt.Sprintf("%s:%s.json", name, i.Tag)
					fileName = filepath.Join(out, prefix+strings.ReplaceAll(fileName, "/", "-"))
					b, err := json.MarshalIndent(r, "", "  ")
					if err != nil {
						return err
					}
					if err := os.WriteFile(fileName, b, os.ModePerm); err != nil {
						return err
					}

					_ = bar.Add(1)
				}
			}
		}
		return nil
	}(o.ReportsFolder, "postscan-")
	if err != nil {
		return err
	}
	_ = bar.Finish()

	return nil
}
