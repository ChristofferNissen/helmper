package registry

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/distribution/reference"
	"golang.org/x/xerrors"
)

func RefToImage(r string) (Image, error) {
	ref, err := reference.ParseAnyReference(r)
	if err != nil {
		return Image{}, err
	}

	img := Image{}
	switch r := ref.(type) {
	case reference.Canonical:
		d := reference.Domain(r)
		p := reference.Path(r)

		img.Registry = d
		img.Repository = p
		img.Digest = r.Digest().String()
		img.UseDigest = true

		if t, ok := r.(reference.Tagged); ok {
			img.Tag = t.Tag()
		}

		return img, nil

	case reference.NamedTagged:
		d := reference.Domain(r)
		p := reference.Path(r)

		img.Registry = d
		img.Repository = p
		img.Tag = r.Tag()
		img.UseDigest = false

		return img, nil

	}

	return img, xerrors.New("Image reference not understood")
}

type Image struct {
	Registry   string
	Repository string
	Tag        string
	Digest     string
	UseDigest  bool
	Patch      *bool
}

func (i Image) TagOrDigest() (string, error) {
	switch {
	case i.Tag != "" && i.Digest != "":
		return fmt.Sprintf("%s@%s", i.Tag, i.Digest), nil
	case i.Tag == "" && i.Digest != "":
		return i.Digest, nil
	case i.Tag != "" && i.Digest == "":
		return i.Tag, nil
	default:
		return "", xerrors.Errorf("No tag or digest")
	}
}

func (i Image) String() (string, error) {
	ref := filepath.Join([]string{i.Registry, i.Repository}...)

	// Append tag value
	if i.Tag != "" {
		ref = fmt.Sprintf("%s:%s", ref, i.Tag)
	}

	// Append digest value
	if i.Digest != "" {
		ref = fmt.Sprintf("%s@%s", ref, i.Digest)
	}

	res, err := reference.ParseAnyReference(ref)
	if err != nil {
		return "", err
	}

	return res.String(), nil
}

func (i Image) Elements() (string, string, string) {

	ref, _ := i.String()
	// if err != nil {
	// 	return "", err
	// }

	res, _ := reference.ParseAnyReference(ref)
	// if err != nil {
	// 	return "", err
	// }

	switch r := res.(type) {
	case reference.Named:
		withoutDomain := strings.Replace(r.Name(), reference.Domain(r)+"/", "", 1)
		repository, name := strings.Split(withoutDomain, "/")[0], strings.Split(withoutDomain, "/")[1]
		return reference.Domain(r), repository, name
	default:
		return "", "", ""
	}
}

func (i Image) ImageName() (string, error) {
	// "github.com/distribution/reference"
	ref, err := i.String()
	if err != nil {
		return "", err
	}

	res, err := reference.ParseAnyReference(ref)
	if err != nil {
		return "", err
	}

	switch r := res.(type) {
	case reference.Named:
		withoutDomain := strings.Replace(r.Name(), reference.Domain(r)+"/", "", 1)
		return withoutDomain, nil
	default:
		return "", xerrors.Errorf("Image could not be parsed")
	}
}

func (i *Image) In(s []Image) bool {
	for _, e := range s {
		if i.Registry == e.Registry && i.Repository == e.Repository && i.Tag == e.Tag {
			return true
		}
	}
	return false
}
