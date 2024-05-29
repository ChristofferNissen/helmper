package cosign

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ChristofferNissen/helmper/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/sign"
)

type SignOption struct {
	Imgs       []*registry.Image
	Registries []registry.Registry

	KeyRef            string
	KeyRefPass        string
	AllowInsecure     bool
	AllowHTTPRegistry bool
}

// cosignAdapter wraps the cosign CLIs native code
func (so SignOption) Run() error {

	// Return early i no images to sign, or no registries to upload signature to
	if !(len(so.Imgs) > 0) || !(len(so.Registries) >= 0) {
		slog.Debug("No images or registries specified. Skipping signing images...")
		return nil
	}

	bar := progressbar.NewOptions(len(so.Imgs), progressbar.OptionSetWriter(ansi.NewAnsiStdout()), // "github.com/k0kubun/go-ansi"
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetDescription("Signing images...\r"),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	// Sign with cosign

	ro := options.RootOptions{
		Timeout: 10 * time.Second,
		Verbose: false,
	}
	ko := options.KeyOpts{
		KeyRef:   so.KeyRef,
		PassFunc: func(bool) ([]byte, error) { return []byte(so.KeyRefPass), nil },
	}
	signOpts := options.SignOptions{
		Upload:           true,
		TlogUpload:       false,
		SkipConfirmation: true,

		Registry: options.RegistryOptions{
			AllowInsecure:     so.AllowInsecure,
			AllowHTTPRegistry: so.AllowHTTPRegistry,
			RegistryClientOpts: []remote.Option{
				remote.WithRetryBackoff(remote.Backoff{
					Duration: 1 * time.Second,
					Jitter:   1.0,
					Factor:   2.0,
					Steps:    5,
					Cap:      2 * time.Minute,
				}),
			},
		},
	}

	for _, r := range so.Registries {
		refs := []string{}
		for _, i := range so.Imgs {
			name, _ := i.ImageName()
			ref := fmt.Sprintf("%s/%s@%s", r.URL, name, i.Digest)
			refs = append(refs, ref)
		}
		if err := sign.SignCmd(&ro, ko, signOpts, refs); err != nil {
			return err
		}
		_ = bar.Add(len(refs))
	}

	_ = bar.Finish()

	return nil
}
