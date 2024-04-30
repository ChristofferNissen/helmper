package cosign

import (
	"fmt"
	"os"
	"time"

	"github.com/ChristofferNissen/helmper/helmper/pkg/registry"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/sigstore/cosign/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/cmd/cosign/cli/sign"
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
		NoTlogUpload:     false,
		SkipConfirmation: true,

		Registry: options.RegistryOptions{
			AllowInsecure:     so.AllowInsecure,
			AllowHTTPRegistry: so.AllowHTTPRegistry,
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
