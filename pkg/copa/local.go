package copa

import (
	"context"
	"fmt"
	"log"

	"time"

	"github.com/ChristofferNissen/helmper/pkg/image"
	myBar "github.com/ChristofferNissen/helmper/pkg/util/bar"
	"github.com/project-copacetic/copacetic/pkg/buildkit"
)

type LocalImage struct {
	Ref  string
	Path string
}

type LocalPatchOption struct {
	Refs []LocalImage

	TarFolder    string
	ReportFolder string

	Buildkit struct {
		Addr       string
		CACertPath string
		CertPath   string
		KeyPath    string
	}

	IgnoreErrors bool
	Architecture *string
}

func (o LocalPatchOption) Run(ctx context.Context, reportFilePaths map[string]string, outFilePaths map[string]string) error {

	bar := myBar.New("Patching images...\r", len(o.Refs))

	seenImages := []image.Image{}
	// for _, m := range o.Data {
	// 	for i := range m {
	for _, r := range o.Refs {
		i, _ := image.RefToImage(r.Ref)
		if i.In(seenImages) {
			log.Printf("Already patched '%s', skipping...\n", r.Ref)
			continue
		}
		// make sure we don't parse again
		seenImages = append(seenImages, i)

		if err := PatchLocal(ctx, 30*time.Minute, r.Ref, r.Path, reportFilePaths[r.Ref], i.Tag, "", "trivy", "openvex", "", o.IgnoreErrors, buildkit.Opts{
			Addr:       o.Buildkit.Addr,
			CACertPath: o.Buildkit.CACertPath,
			CertPath:   o.Buildkit.CertPath,
			KeyPath:    o.Buildkit.KeyPath,
		}, outFilePaths[r.Ref]); err != nil {
			return fmt.Errorf("error patching image %s :: %w ", r.Ref, err)
		}

		_ = bar.Add(1)
		// }
	}
	_ = bar.Finish()

	// bar = myBar.New("Pushing images from tar...\r", size)
	// for r, m := range o.Data {
	// 	for i, b := range m {
	// 		if b {
	// 			name, _ := i.ImageName()

	// 			store, err := oci.NewFromTar(ctx, outFilePaths[i])
	// 			if err != nil {
	// 				return err
	// 			}
	// 			manifest, err := store.Resolve(ctx, i.Tag)
	// 			if err != nil {
	// 				return err
	// 			}
	// 			i.Digest = manifest.Digest.String()

	// 			if r.PrefixSource {
	// 				old := name
	// 				name, _ = image.UpdateNameWithPrefixSource(i)
	// 				slog.Info("registry has PrefixSource enabled", slog.String("old", old), slog.String("new", name))
	// 			}

	// 			// Connect to a remote repository
	// 			url, _ := strings.CutPrefix(r.URL, "oci://")
	// 			repo, err := remote.NewRepository(url + "/" + name)
	// 			if err != nil {
	// 				return err
	// 			}

	// 			repo.PlainHTTP = r.PlainHTTP

	// 			// Prepare authentication using Docker credentials
	// 			storeOpts := credentials.StoreOptions{}
	// 			credStore, err := credentials.NewStoreFromDocker(storeOpts)
	// 			if err != nil {
	// 				return err
	// 			}
	// 			repo.Client = &auth.Client{
	// 				Client:     retry.DefaultClient,
	// 				Cache:      auth.NewCache(),
	// 				Credential: credentials.Credential(credStore), // Use the credentials store
	// 			}

	// 			// Copy from the file store to the remote repository
	// 			opts := oras.DefaultCopyOptions
	// 			if o.Architecture != nil {
	// 				v, err := v1.ParsePlatform(*o.Architecture)
	// 				if err != nil {
	// 					return err
	// 				}
	// 				opts.WithTargetPlatform(
	// 					&v1_spec.Platform{
	// 						Architecture: v.Architecture,
	// 						OS:           v.OS,
	// 						OSVersion:    v.OSVersion,
	// 						OSFeatures:   v.OSFeatures,
	// 						Variant:      v.Variant,
	// 					},
	// 				)
	// 			}
	// 			manifest, err = oras.Copy(ctx, store, i.Tag, repo, i.Tag, opts)
	// 			if err != nil {
	// 				return err
	// 			}

	// 			i.Digest = manifest.Digest.String()
	// 		}
	// 		_ = bar.Add(1)
	// 	}
	// }

	// _ = bar.Finish()

	return nil
}
