package helm

import (
	"testing"

	"helm.sh/helm/v3/pkg/repo"
)

func TestResolveVersions(t *testing.T) {

	c := ChartCollection{
		Charts: []Chart{
			{
				Name:    "argo-cd",
				Version: ">4.0.0 <5.0.0",
				Repo: repo.Entry{
					Name: "argoproj",
					URL:  "https://argoproj.github.io/argo-helm",
				},
			},
		},
	}

	co := ChartOption{
		ChartCollection: &c,
	}
	_, err := co.ChartCollection.SetupHelm()
	if err != nil {
		t.Error(err)
	}

	vs, err := c.Charts[0].ResolveVersions()
	if err != nil {
		t.Errorf("want '%s' goChartt '%s'", "nil", err.Error())
	}

	if len(vs) != 63 {
		t.Errorf("want '%s' got '%d'", "63", len(vs))
	}

}

func TestResolveVersions2(t *testing.T) {

	c := ChartCollection{
		Charts: []Chart{
			{
				Name:    "argo-cd",
				Version: ">5.51.0 <6.0.0",
				Repo: repo.Entry{
					Name: "argoproj",
					URL:  "https://argoproj.github.io/argo-helm",
				},
			},
		},
	}

	co := ChartOption{
		ChartCollection: &c,
	}
	_, err := co.ChartCollection.SetupHelm()
	if err != nil {
		t.Error(err)
	}

	vs, err := c.Charts[0].ResolveVersions()
	if err != nil {
		t.Errorf("want '%s' got '%s'", "err", "nil")
	}

	if len(vs) != 26 {
		t.Errorf("want '%s' got '%d'", "26", len(vs))
	}

}
