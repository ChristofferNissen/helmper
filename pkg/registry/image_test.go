package registry

import (
	"testing"

	"golang.org/x/xerrors"
)

func testBed() []Image {

	images := []Image{
		{
			Repository: "hello-world",
			Tag:        "latest",
		},
		{
			Registry:   "quay.io",
			Repository: "argoproj/argocd",
			Tag:        "v2.10.0",
		},
		{
			Repository: "ghcr.io/kubereboot/kured",
			Tag:        "1.14.1",
		},
		{
			Repository: "ghcr.io/kubereboot/kured",
			Digest:     "sha256:ca4ae4f37d71a4110889fc4add3c4abef8b96fa6ed977ed399d9b1c3bd7e608e",
		},
		{
			Repository: "ghcr.io/kubereboot/kured",
		},
		{
			Repository: "ghcr.io/kubereboot/kured",
			Tag:        "1.14.1",
			Digest:     "sha256:ca4ae4f37d71a4110889fc4add3c4abef8b96fa6ed977ed399d9b1c3bd7e608e",
		},
		{
			Repository: "public.ecr.aws/eks-distro/kubernetes-csi/livenessprobe",
			Tag:        "v2.13.0-eks-1-30-8",
		},
	}

	return images
}

func TestTagOrDigest(t *testing.T) {
	imgs := testBed()

	e4 := "1.14.1"
	a4, _ := imgs[2].TagOrDigest()
	if a4 != e4 {
		t.Errorf("want '%s' got '%s'", e4, a4)
	}

	e4 = "sha256:ca4ae4f37d71a4110889fc4add3c4abef8b96fa6ed977ed399d9b1c3bd7e608e"
	a4, _ = imgs[3].TagOrDigest()
	if a4 != e4 {
		t.Errorf("want '%s' got '%s'", e4, a4)
	}

	e4 = ""
	e5 := xerrors.Errorf("No tag or digest").Error()
	a4, err := imgs[4].TagOrDigest()
	if a4 != e4 {
		t.Errorf("want '%s' got '%s'", e4, a4)
	}
	if err.Error() != e5 {
		t.Errorf("want '%s' got '%s'", e5, err)
	}

	e4 = "1.14.1@sha256:ca4ae4f37d71a4110889fc4add3c4abef8b96fa6ed977ed399d9b1c3bd7e608e"
	a4, _ = imgs[5].TagOrDigest()
	if a4 != e4 {
		t.Errorf("want '%s' got '%s'", e4, a4)
	}
}

func TestString(t *testing.T) {
	imgs := testBed()

	expected := "docker.io/library/hello-world:latest"
	actual, _ := imgs[0].String()
	if actual != expected {
		t.Errorf("want '%s' got '%s'", expected, actual)
	}

	expected = "quay.io/argoproj/argocd:v2.10.0"
	actual, _ = imgs[1].String()
	if actual != expected {
		t.Errorf("want '%s' got '%s'", expected, actual)
	}

	expected = "ghcr.io/kubereboot/kured:1.14.1"
	actual, _ = imgs[2].String()
	if actual != expected {
		t.Errorf("want '%s' got '%s'", expected, actual)
	}
}

func TestElements(t *testing.T) {
	imgs := testBed()

	e1 := "docker.io"
	e2 := "library"
	e3 := "hello-world"
	e4 := "latest"
	a1, a2, a3 := imgs[0].Elements()
	a4 := imgs[0].Tag
	if a1 != e1 {
		t.Errorf("want '%s' got '%s'", e1, a1)
	}
	if a2 != e2 {
		t.Errorf("want '%s' got '%s'", e2, a2)
	}
	if a3 != e3 {
		t.Errorf("want '%s' got '%s'", e3, a3)
	}
	if a4 != e4 {
		t.Errorf("want '%s' got '%s'", e4, a4)
	}

	e1 = "quay.io"
	e2 = "argoproj"
	e3 = "argocd"
	e4 = "v2.10.0"
	a1, a2, a3 = imgs[1].Elements()
	a4 = imgs[1].Tag
	if a1 != e1 {
		t.Errorf("want '%s' got '%s'", e1, a1)
	}
	if a2 != e2 {
		t.Errorf("want '%s' got '%s'", e2, a2)
	}
	if a3 != e3 {
		t.Errorf("want '%s' got '%s'", e3, a3)
	}
	if a4 != e4 {
		t.Errorf("want '%s' got '%s'", e4, a4)
	}

	e1 = "ghcr.io"
	e2 = "kubereboot"
	e3 = "kured"
	e4 = "1.14.1"
	a1, a2, a3 = imgs[2].Elements()
	a4, _ = imgs[2].TagOrDigest()
	if a1 != e1 {
		t.Errorf("want '%s' got '%s'", e1, a1)
	}
	if a2 != e2 {
		t.Errorf("want '%s' got '%s'", e2, a2)
	}
	if a3 != e3 {
		t.Errorf("want '%s' got '%s'", e3, a3)
	}
	if a4 != e4 {
		t.Errorf("want '%s' got '%s'", e4, a4)
	}

	e1 = "ghcr.io"
	e2 = "kubereboot"
	e3 = "kured"
	e4 = "sha256:ca4ae4f37d71a4110889fc4add3c4abef8b96fa6ed977ed399d9b1c3bd7e608e"
	a1, a2, a3 = imgs[3].Elements()
	a4, _ = imgs[3].TagOrDigest()
	if a1 != e1 {
		t.Errorf("want '%s' got '%s'", e1, a1)
	}
	if a2 != e2 {
		t.Errorf("want '%s' got '%s'", e2, a2)
	}
	if a3 != e3 {
		t.Errorf("want '%s' got '%s'", e3, a3)
	}
	if a4 != e4 {
		t.Errorf("want '%s' got '%s'", e4, a4)
	}

	e1 = "public.ecr.aws"
	e2 = "eks-distro/kubernetes-csi"
	e3 = "livenessprobe"
	e4 = "v2.13.0-eks-1-30-8"
	a1, a2, a3 = imgs[6].Elements()
	a4, _ = imgs[6].TagOrDigest()
	if a1 != e1 {
		t.Errorf("want '%s' got '%s'", e1, a1)
	}
	if a2 != e2 {
		t.Errorf("want '%s' got '%s'", e2, a2)
	}
	if a3 != e3 {
		t.Errorf("want '%s' got '%s'", e3, a3)
	}
	if a4 != e4 {
		t.Errorf("want '%s' got '%s'", e4, a4)
	}

}

func TestImageName(t *testing.T) {
	imgs := testBed()

	expected := "library/hello-world"
	actual, _ := imgs[0].ImageName()
	if actual != expected {
		t.Errorf("want '%s' got '%s'", expected, actual)
	}

	expected = "argoproj/argocd"
	actual, _ = imgs[1].ImageName()
	if actual != expected {
		t.Errorf("want '%s' got '%s'", expected, actual)
	}

	expected = "kubereboot/kured"
	actual, _ = imgs[2].ImageName()
	if actual != expected {
		t.Errorf("want '%s' got '%s'", expected, actual)
	}

	expected = "kubereboot/kured"
	actual, _ = imgs[3].ImageName()
	if actual != expected {
		t.Errorf("want '%s' got '%s'", expected, actual)
	}
}
