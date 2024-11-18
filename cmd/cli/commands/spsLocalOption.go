package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChristofferNissen/helmper/pkg/copa"
	"github.com/ChristofferNissen/helmper/pkg/trivy"
	myBar "github.com/ChristofferNissen/helmper/pkg/util/bar"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
)

type SpsLocalOption struct {
	// Data          map[*registry.Registry]map[*image.Image]bool
	Data []importEntry

	All           bool
	Architecture  *string
	ReportsFolder string
	ReportsClean  bool
	TarsFolder    string
	TarsClean     bool
	ScanOption    trivy.ScanOption
	PatchOption   copa.LocalPatchOption
}

func (o SpsLocalOption) Run(ctx context.Context) error {
	// // Count of images to import across registries
	// lenImages := func() int {
	// 	c := 0
	// 	seen := make([]image.Image, 0)
	// 	for _, m := range o.Data {
	// 		for i, b := range m {
	// 			if b {
	// 				if i.In(seen) {
	// 					continue
	// 				}
	// 				seen = append(seen, *i)
	// 				c++
	// 			}
	// 		}
	// 	}
	// 	return c
	// }()

	// if !(lenImages > 0) {
	// 	return nil
	// }

	lenImages := len(o.Data)

	// Trivy scan
	bar := myBar.New("Scanning images before patching...\r", lenImages)
	patch, err := func() ([]*importEntry, error) {
		// imgs := make([]*image.Image, 0)
		// for _, m := range o.Data {
		// 	for i, b := range m {
		// 		if b {
		// 			if i.InP(imgs) {
		// 				continue
		// 			}
		// 			imgs = append(imgs, i)
		// 		}
		// 	}
		// }

		patch := make([]*importEntry, 0)
		for _, i := range o.Data {

			if i.Type == "Helm Chart" {
				// Helm charts can't be patched, but could be scanned..
				continue
			}

			var ociLayoutPath string
			if strings.HasSuffix(i.FilePath, ".tar") {
				store, err := oci.NewFromTar(ctx, "/tmp/helmper-export/"+i.FilePath)
				if err != nil {
					return nil, err
				}

				tempDir, err := os.MkdirTemp("", "oras_oci_example_*")
				if err != nil {
					panic(err) // Handle error
				}
				defer os.RemoveAll(tempDir)
				target, err := oci.New(tempDir)
				if err != nil {
					panic(err) // Handle error
				}
				if _, err := oras.Copy(ctx, store, i.Tag, target, i.Tag, oras.DefaultCopyOptions); err != nil {
					return nil, err
				}

				ociLayoutPath = tempDir

			} else {
				// oci-layout not implemented
				continue
			}
			// cleanup := func() {
			// 	os.RemoveAll("/tmp/helmper-export/" + i.FilePath)
			// }
			// defer cleanup()

			// if i.Patch != nil {
			// 	if !*i.Patch {
			// 		ref := i.String()
			// 		slog.Debug("image should not be patched", slog.String("image", ref))
			// 		push = append(push, i)
			// 		continue
			// 	}
			// }

			// log.Println(i.FilePath)
			// ref := i.String()

			r, err := o.ScanOption.Scan(ociLayoutPath)
			if err != nil {
				return nil, err
			}

			switch copa.SupportedOS(r.Metadata.OS) {
			case true:
				// filter images with no os-pkgs as copa has nothing to do
				switch trivy.ContainsOsPkgs(r.Results) {
				case true:
					slog.Debug("Image does contain os-pkgs vulnerabilities",
						slog.String("image", i.Name))
					patch = append(patch, &i)
				case false:
					slog.Warn("Image does not contain os-pkgs. The image will not be patched.",
						slog.String("image", i.Name),
						slog.Any("results", r.Results),
					)
					// 	push = append(push, i)
				}
			case false:
				slog.Warn("Image contains an unsupported OS. The image will not be patched.",
					slog.String("os", fmt.Sprintf("%v", r.Metadata.OS)),
					slog.String("image", i.Name),
					slog.Any("results", r.Results),
				)
				// push = append(push, i)true
			}

			// Write report to filesystem
			name := i.Name
			fileName := fmt.Sprintf("%s:%s.json", name, i.Tag)
			fileName = filepath.Join(o.ReportsFolder, "prescan-"+strings.ReplaceAll(fileName, "/", "-"))
			b, err := json.MarshalIndent(r, "", "  ")
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(fileName, b, os.ModePerm); err != nil {
				return nil, err
			}
		}

		return patch, nil

		// filter images
		// patchM := make(map[*registry.Registry]map[*image.Image]bool, 0)
		// pushM := make(map[*registry.Registry]map[*image.Image]bool, 0)
		// for r, elem := range o.Data {
		// 	patchRegistry := patchM[r]
		// 	if patchRegistry == nil {
		// 		patchRegistry = make(map[*image.Image]bool, 0)
		// 		patchM[r] = patchRegistry
		// 	}
		// 	pushRegistry := pushM[r]
		// 	if pushRegistry == nil {
		// 		pushRegistry = make(map[*image.Image]bool, 0)
		// 		pushM[r] = pushRegistry
		// 	}

		// 	for i, b := range elem {
		// 		if b {
		// 			switch {
		// 			case i.InP(patch):
		// 				patchRegistry[i] = true
		// 			case i.InP(push):
		// 				pushRegistry[i] = true
		// 			}

		// 			_ = bar.Add(1)
		// 		}
		// 	}

		// 	patchM[r] = patchRegistry
		// 	pushM[r] = pushRegistry
		// }

		// return patchM, pushM, nil
	}()
	if err != nil {
		return err
	}
	_ = bar.Finish()

	// Determine fully qualified output path for images
	reportFilePaths := make(map[string]string)
	reportPostFilePaths := make(map[string]string)
	outFilePaths := make(map[string]string)
	for _, i := range patch {
		name := i.Name
		fileName := fmt.Sprintf("prescan-%s:%s.json", name, i.Tag)
		reportFilePaths[i.Registry+"/"+i.Name+":"+i.Tag] = filepath.Join(
			o.ReportsFolder,
			strings.ReplaceAll(fileName, "/", "-"),
		)
		fileName = fmt.Sprintf("postscan-%s:%s.json", name, i.Tag)
		reportPostFilePaths[i.Registry+"/"+i.Name+":"+i.Tag] = filepath.Join(
			o.ReportsFolder,
			strings.ReplaceAll(fileName, "/", "-"),
		)
		out := fmt.Sprintf("%s:%s.tar", name, i.Tag)
		outFilePaths[i.Registry+"/"+i.Name+":"+i.Tag] = filepath.Join(
			o.TarsFolder,
			strings.ReplaceAll(out, "/", "-"),
		)
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
	// io := registry.ImportOption{
	// 	Data:         push,
	// 	All:          o.All,
	// 	Architecture: o.Architecture,
	// }
	// err = io.Run(context.WithoutCancel(ctx))
	// if err != nil {
	// 	return err
	// }

	li := make([]copa.LocalImage, 0)
	for _, e := range patch {
		li = append(li, copa.LocalImage{
			Ref:  e.Registry + "/" + e.Name + ":" + e.Tag,
			Path: "/tmp/helmper-export/" + e.FilePath,
		})
	}

	// log.Println(li)

	// Patch image and save to tar
	// o.PatchOption.Data = patch
	o.PatchOption.Refs = li
	err = o.PatchOption.Run(ctx, reportFilePaths, outFilePaths)
	if err != nil {
		return err
	}

	// bar = myBar.New("Scanning images after patching...\r", lenImages)
	// err = func(out string, prefix string) error {
	// 	for _, m := range o.Data {
	// 		for i, b := range m {
	// 			if b {
	// 				ref := i.String()

	// 				slog.Default().With(slog.String("image", ref))

	// 				r, err := o.ScanOption.Scan(ref)
	// 				if err != nil {
	// 					return err
	// 				}

	// 				// Write report to filesystem
	// 				name, _ := i.ImageName()
	// 				fileName := fmt.Sprintf("%s:%s.json", name, i.Tag)
	// 				fileName = filepath.Join(out, prefix+strings.ReplaceAll(fileName, "/", "-"))
	// 				b, err := json.MarshalIndent(r, "", "  ")
	// 				if err != nil {
	// 					return err
	// 				}
	// 				if err := os.WriteFile(fileName, b, os.ModePerm); err != nil {
	// 					return err
	// 				}

	// 				_ = bar.Add(1)
	// 			}
	// 		}
	// 	}
	// 	return nil
	// }(o.ReportsFolder, "postscan-")
	// if err != nil {
	// 	return err
	// }
	// _ = bar.Finish()

	return nil
}
