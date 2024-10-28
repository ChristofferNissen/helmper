package image

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/distribution/reference"
)

// UpdateNameWithPrefixSource updates the image name with the registry prefix if PrefixSource is enabled.
func UpdateNameWithPrefixSource(i *Image) (string, error) {
	name, err := i.ImageName()
	if err != nil {
		return "", err
	}
	reg, _, _, err := i.Elements()
	if err != nil {
		return "", err
	}
	noPorts := strings.SplitN(reg, ":", 2)[0]
	noTLD := strings.SplitN(noPorts, ".", 2)[0]
	return noTLD + "/" + name, nil
}

// RefToImage parses the reference string and returns an Image.
func RefToImage(r string) (Image, error) {
	ref, err := reference.ParseAnyReference(r)
	if err != nil {
		return Image{}, fmt.Errorf("failed to parse reference: %w", err)
	}

	img := Image{}

	switch r := ref.(type) {
	case reference.Canonical:
		img.Registry = reference.Domain(r)
		img.Repository = reference.Path(r)
		img.Digest = r.Digest().String()
		img.UseDigest = true
		if t, ok := r.(reference.Tagged); ok {
			img.Tag = t.Tag()
		}
	case reference.NamedTagged:
		img.Registry = reference.Domain(r)
		img.Repository = reference.Path(r)
		img.Tag = r.Tag()
		img.UseDigest = false
	default:
		return img, fmt.Errorf("image reference not understood")
	}

	return img, nil
}

type Image struct {
	Registry   string
	Repository string
	Tag        string
	Digest     string
	UseDigest  bool
	Patch      *bool
	parsedRef  *string
}

// IsEmpty determines if an image is empty (i.e., registry, repository, and name are empty).
func (i Image) IsEmpty() bool {
	return i.Registry == "" && i.Repository == "" && i.Tag == ""
}

// TagOrDigest returns a string representation of either the tag or digest.
func (i Image) TagOrDigest() (string, error) {
	switch {
	case i.Tag != "" && i.Digest != "":
		return fmt.Sprintf("%s@%s", i.Tag, i.Digest), nil
	case i.Tag == "" && i.Digest != "":
		return i.Digest, nil
	case i.Tag != "" && i.Digest == "":
		return i.Tag, nil
	default:
		return "", fmt.Errorf("no tag or digest")
	}
}

// String returns the string representation of the image.

func cleanString(s string, remove string) string {
	noSuffix, _ := strings.CutSuffix(s, remove)
	noPrefix, _ := strings.CutPrefix(noSuffix, remove)
	return noPrefix
}

func (i *Image) String() string {
	if i.parsedRef != nil {
		return *i.parsedRef
	}

	var refBuilder strings.Builder

	// Join the registry and repository
	if i.Registry != "" {
		refBuilder.WriteString(cleanString(i.Registry, "/"))
		refBuilder.WriteString("/")
	}
	refBuilder.WriteString(cleanString(i.Repository, "/"))

	// Append tag if present
	if i.Tag != "" {
		refBuilder.WriteString(":")
		refBuilder.WriteString(cleanString(i.Tag, ":"))
	}

	// Append digest if needed
	if i.UseDigest && i.Digest != "" {
		refBuilder.WriteString("@")
		refBuilder.WriteString(cleanString(i.Digest, "@"))
	}

	ref := refBuilder.String()
	res, err := reference.ParseAnyReference(ref)
	if err != nil {
		return ref
	}

	if !strings.HasPrefix(res.String(), i.Registry) {
		return ref
	}

	s := res.String()
	i.parsedRef = to.Ptr(s)
	return s
}

// Elements returns the registry, repository, and name of the image.
func (i Image) Elements() (string, string, string, error) {
	ref := i.String()
	res, err := reference.ParseNamed(ref)
	if err != nil {
		return "", "", "", err
	}

	switch r := res.(type) {
	case reference.Named:
		withoutDomain := strings.TrimPrefix(r.Name(), reference.Domain(r)+"/")
		parts := strings.Split(withoutDomain, "/")
		var repository, name string
		if len(parts) == 2 {
			repository = parts[0]
			name = parts[1]
		} else if len(parts) > 2 {
			repository = strings.Join(parts[:2], "/")
			name = strings.Join(parts[2:], "/")
		} else {
			repository = "library"
			name = parts[0]
		}
		return reference.Domain(r), repository, name, nil
	default:
		return "", "", "", fmt.Errorf("failed to parse elements")
	}
}

// ImageName returns the full name of the image.
func (i Image) ImageName() (string, error) {
	res, err := reference.ParseNamed(i.String())
	if err != nil {
		return "", fmt.Errorf("failed to parse reference: %w", err)
	}
	switch r := res.(type) {
	case reference.Named:
		withoutDomain := strings.TrimPrefix(r.Name(), reference.Domain(r)+"/")
		return withoutDomain, nil
	default:
		return "", fmt.Errorf("image could not be parsed")
	}
}

// In checks if the image is in a slice of images.
func (i *Image) In(s []Image) bool {
	for _, e := range s {
		if i.Registry == e.Registry && i.Repository == e.Repository && i.Tag == e.Tag {
			return true
		}
	}
	return false
}

// InP checks if the image is in a slice of pointers to images.
func (i *Image) InP(s []*Image) bool {
	for _, e := range s {
		if i.Registry == e.Registry && i.Repository == e.Repository && i.Tag == e.Tag {
			return true
		}
	}
	return false
}
