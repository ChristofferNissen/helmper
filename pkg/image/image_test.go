package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
)

func testBed() []Image {
	return []Image{
		{Repository: "hello-world", Tag: "latest"},
		{Registry: "quay.io", Repository: "argoproj/argocd", Tag: "v2.10.0"},
		{Repository: "ghcr.io/kubereboot/kured", Tag: "1.14.1"},
		{Repository: "ghcr.io/kubereboot/kured", Digest: "sha256:ca4ae4f37d71a4110889fc4add3c4abef8b96fa6ed977ed399d9b1c3bd7e608e"},
		{Repository: "ghcr.io/kubereboot/kured"},
		{Repository: "ghcr.io/kubereboot/kured", Tag: "1.14.1", Digest: "sha256:ca4ae4f37d71a4110889fc4add3c4abef8b96fa6ed977ed399d9b1c3bd7e608e"},
		{Registry: "public.ecr.aws", Repository: "eks-distro/kubernetes-csi/livenessprobe", Tag: "v2.13.0-eks-1-30-8"},
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		img      Image
		expected bool
	}{
		{Image{Registry: "", Repository: "", Tag: ""}, true},
		{Image{Registry: "docker.io", Repository: "", Tag: ""}, false},
		{Image{Registry: "", Repository: "library/hello-world", Tag: ""}, false},
		{Image{Registry: "", Repository: "", Tag: "latest"}, false},
		{Image{Registry: "docker.io", Repository: "library/hello-world", Tag: "latest"}, false},
	}

	for _, tt := range tests {
		result := tt.img.IsEmpty()
		assert.Equal(t, tt.expected, result)
	}
}

func TestUpdateNameWithPrefixSource(t *testing.T) {
	img := Image{Repository: "library/hello-world"}
	expected := "docker/library/hello-world"
	result, err := UpdateNameWithPrefixSource(&img)
	assert.NoError(t, err, "expected no error")
	assert.Equal(t, expected, result)
}

func TestRefToImage(t *testing.T) {
	ref := "docker.io/library/hello-world:latest"
	img, err := RefToImage(ref)
	assert.NoError(t, err)
	assert.Equal(t, "docker.io", img.Registry)
	assert.Equal(t, "library/hello-world", img.Repository)
	assert.Equal(t, "latest", img.Tag)
}

func TestTagOrDigest(t *testing.T) {
	imgs := testBed()

	tests := []struct {
		img      Image
		expected string
		wantErr  bool
	}{
		{imgs[2], "1.14.1", false},
		{imgs[3], "sha256:ca4ae4f37d71a4110889fc4add3c4abef8b96fa6ed977ed399d9b1c3bd7e608e", false},
		{imgs[4], "", true},
		{imgs[5], "1.14.1@sha256:ca4ae4f37d71a4110889fc4add3c4abef8b96fa6ed977ed399d9b1c3bd7e608e", false},
	}

	for _, tt := range tests {
		result, err := tt.img.TagOrDigest()
		if tt.wantErr {
			assert.Error(t, err, "expected an error")
			assert.Equal(t, xerrors.Errorf("no tag or digest").Error(), err.Error())
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		}
	}
}

func TestString(t *testing.T) {
	imgs := testBed()

	tests := []struct {
		img      Image
		expected string
	}{
		{imgs[0], "docker.io/library/hello-world:latest"},
		{imgs[1], "quay.io/argoproj/argocd:v2.10.0"},
		{imgs[2], "ghcr.io/kubereboot/kured:1.14.1"},
	}

	for _, tt := range tests {
		result := tt.img.String()
		// assert.NoError(t, err)
		assert.Equal(t, tt.expected, result)
	}
}

func TestElements(t *testing.T) {
	imgs := testBed()

	tests := []struct {
		img                 Image
		expectedReg         string
		expectedRepo        string
		expectedName        string
		expectedTagOrDigest string
	}{
		{imgs[0], "docker.io", "library", "hello-world", "latest"},
		{imgs[1], "quay.io", "argoproj", "argocd", "v2.10.0"},
		{imgs[2], "ghcr.io", "kubereboot", "kured", "1.14.1"},
		{imgs[3], "ghcr.io", "kubereboot", "kured", "sha256:ca4ae4f37d71a4110889fc4add3c4abef8b96fa6ed977ed399d9b1c3bd7e608e"},
		{imgs[6], "public.ecr.aws", "eks-distro/kubernetes-csi", "livenessprobe", "v2.13.0-eks-1-30-8"},
	}

	for _, tt := range tests {
		reg, repo, name, _ := tt.img.Elements()
		tagOrDigest, err := tt.img.TagOrDigest()
		assert.NoError(t, err, "expected no errors")
		assert.Equal(t, tt.expectedReg, reg)
		assert.Equal(t, tt.expectedRepo, repo)
		assert.Equal(t, tt.expectedName, name)
		assert.Equal(t, tt.expectedTagOrDigest, tagOrDigest)
	}
}

func TestImageName(t *testing.T) {
	imgs := testBed()

	tests := []struct {
		img      Image
		expected string
	}{
		{imgs[0], "library/hello-world"},
		{imgs[1], "argoproj/argocd"},
		{imgs[2], "kubereboot/kured"},
		{imgs[3], "kubereboot/kured"},
	}

	for _, tt := range tests {
		result, err := tt.img.ImageName()
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, result)
	}
}

func TestIn(t *testing.T) {
	img := Image{Registry: "docker.io", Repository: "library/hello-world", Tag: "latest"}
	images := []Image{
		{Registry: "docker.io", Repository: "library/hello-world", Tag: "latest"},
		{Registry: "quay.io", Repository: "argoproj/argocd", Tag: "v2.10.0"},
	}

	result := img.In(images)
	assert.True(t, result)

	imgNotExist := Image{Registry: "docker.io", Repository: "library/hello-world", Tag: "1.0"}
	result = imgNotExist.In(images)
	assert.False(t, result)
}

func TestInP(t *testing.T) {
	img := Image{Registry: "docker.io", Repository: "library/hello-world", Tag: "latest"}
	images := []*Image{
		{Registry: "docker.io", Repository: "library/hello-world", Tag: "latest"},
		{Registry: "quay.io", Repository: "argoproj/argocd", Tag: "v2.10.0"},
	}

	result := img.InP(images)
	assert.True(t, result)

	imgNotExist := Image{Registry: "docker.io", Repository: "library/hello-world", Tag: "1.0"}
	result = imgNotExist.InP(images)
	assert.False(t, result)
}

func TestImageNameError(t *testing.T) {
	img := Image{Registry: "invalid_registry", Repository: "repo"}
	res, err := img.ImageName()
	assert.Error(t, err, "Expected error for invalid image name")
	assert.Empty(t, res, "expected empty name")
}

func TestElementsError(t *testing.T) {
	img := Image{Registry: "invalid_registry", Repository: "repo"}
	reg, repo, name, err := img.Elements()
	assert.Error(t, err, "Expected error for invalid registry")
	assert.Empty(t, reg, "Expected empty registry")
	assert.Empty(t, repo, "Expected empty repository")
	assert.Empty(t, name, "Expected empty name")
}

func TestRefToImage_ErrorCases(t *testing.T) {
	tests := []struct {
		ref        string
		shouldFail bool
	}{
		{"invalid-reference@", true},
		{"docker.io/library/hello-world@invalid-digest", true},
	}

	for _, tt := range tests {
		_, err := RefToImage(tt.ref)
		if tt.shouldFail {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestUpdateNameWithPrefixSource_ErrorCases(t *testing.T) {
	img := Image{Registry: "invalid_registry", Repository: "repo"}
	result, err := UpdateNameWithPrefixSource(&img)
	assert.Error(t, err, "")
	assert.Empty(t, result)
}

func TestRefToImage_AdditionalErrorCases(t *testing.T) {
	tests := []struct {
		ref        string
		shouldFail bool
	}{
		{"", true},
		{"://invalid-reference", true},
		{"http://docker.io/library/hello-world", true},
	}

	for _, tt := range tests {
		_, err := RefToImage(tt.ref)
		if tt.shouldFail {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestImageTagOrDigest_ErrorCases(t *testing.T) {
	img := Image{}
	result, err := img.TagOrDigest()
	assert.Empty(t, result, "Expected empty result for empty image")
	assert.Error(t, err, "Expected error for empty image")
}

func TestImageString_BoundaryCases(t *testing.T) {
	img := Image{Registry: "docker.io/", Repository: "/library/hello-world", Tag: "latest"}
	result := img.String()
	assert.Contains(t, result, "docker.io/library/hello-world:latest", "Expected correctly formatted string despite leading slash")
}

func TestIn_NullCases(t *testing.T) {
	img := Image{Registry: "docker.io", Repository: "library/hello-world", Tag: "latest"}
	images := []Image{}

	result := img.In(images)
	assert.False(t, result, "Expected false for empty images slice")

	resultP := img.InP(nil)
	assert.False(t, resultP, "Expected false for nil images slice")
}

func TestReplaceRegistry(t *testing.T) {
	tests := []struct {
		img            Image
		newRegistry    string
		expectedString string
	}{
		{
			Image{Registry: "docker.io", Repository: "library/hello-world", Tag: "latest"},
			"quay.io",
			"quay.io/library/hello-world:latest",
		},
		{
			Image{Registry: "gcr.io", Repository: "k8s-artifacts-prod/gce", Tag: "v1.0.0"},
			"eu.gcr.io",
			"eu.gcr.io/k8s-artifacts-prod/gce:v1.0.0",
		},
		{
			Image{Registry: "docker.io", Repository: "library/busybox", Tag: "1.31"},
			"registry.hub.docker.com",
			"registry.hub.docker.com/library/busybox:1.31",
		},
	}

	for _, tt := range tests {
		result := tt.img.ReplaceRegistry(tt.newRegistry)
		assert.Equal(t, tt.expectedString, result)
		assert.Equal(t, tt.newRegistry, tt.img.Registry)
		assert.Equal(t, result, *tt.img.parsedRef)
	}
}
