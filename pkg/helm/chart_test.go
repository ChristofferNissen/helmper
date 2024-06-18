package helm

import (
	"testing"

	"helm.sh/helm/v3/pkg/repo"
)

func TestResolveVersions(t *testing.T) {

	c := Chart{
		Name:    "argo-cd",
		Version: ">4.0.0 <5.0.0",
		Repo: repo.Entry{
			Name: "argoproj",
			URL:  "https://argoproj.github.io/argo-helm",
		},
	}

	c.AddToHelmRepositoryFile()
	c.Pull()

	vs, err := c.ResolveVersions()
	if err != nil {
		t.Errorf("want '%s' got '%s'", "nil", err.Error())
	}

	if len(vs) != 63 {
		t.Errorf("want '%s' got '%d'", "63", len(vs))
	}

}

func TestResolveVersions2(t *testing.T) {

	c := Chart{
		Name:    "argo-cd",
		Version: ">5.51.0 <6.0.0",
		Repo: repo.Entry{
			Name: "argoproj",
			URL:  "https://argoproj.github.io/argo-helm",
		},
	}

	c.AddToHelmRepositoryFile()
	c.Pull()

	vs, err := c.ResolveVersions()
	if err != nil {
		t.Errorf("want '%s' got '%s'", "err", "nil")
	}

	if len(vs) != 26 {
		t.Errorf("want '%s' got '%d'", "26", len(vs))
	}

}
