package helm

import (
	"testing"

	"helm.sh/helm/v3/pkg/repo"
)

func TestResolveVersions(t *testing.T) {

	c := Chart{
		Name:    "argo-cd",
		Version: ">4.0.0 <5.0.0 || >=6.0.0",
		Repo: repo.Entry{
			Name: "argo",
		},
	}

	vs, err := c.ResolveVersions()
	if err != nil {
		t.Errorf("want '%s' got '%s'", "err", "nil")
	}

	if len(vs) != 127 {
		t.Errorf("want '%s' got '%d'", "127", len(vs))
	}

}

func TestResolveVersions2(t *testing.T) {

	c := Chart{
		Name:    "argo-cd",
		Version: ">5.51.0 <6.0.0 || >=7.0.0",
		Repo: repo.Entry{
			Name: "argo",
		},
	}

	vs, err := c.ResolveVersions()
	if err != nil {
		t.Errorf("want '%s' got '%s'", "err", "nil")
	}

	if len(vs) != 31 {
		t.Errorf("want '%s' got '%d'", "31", len(vs))
	}

}
