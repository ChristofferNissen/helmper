package helm

import (
	"os"
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

func TestPull(t *testing.T) {
	cases := []struct {
		name        string
		chart       Chart
		expectErr   bool
		expectExist bool
	}{
		{
			name: "Valid OCI URL",
			chart: Chart{
				Repo: repo.Entry{
					URL: "oci://chartproxy.container-registry.com/charts.jetstack.io/cert-manager",
				},
				Name:    "cert-manager",
				Version: "1.0.0",
			},
			expectErr:   false,
			expectExist: true,
		},
		{
			name: "Valid non-OCI URL",
			chart: Chart{
				Repo: repo.Entry{
					URL:                   "https://kubernetes.github.io/ingress-nginx",
					InsecureSkipTLSverify: false,
					Username:              "",
					Password:              "",
				},
				Name:    "ingress-nginx",
				Version: "4.11.3",
			},
			expectErr:   false,
			expectExist: true,
		},
		{
			name: "Invalid URL",
			chart: Chart{
				Repo: repo.Entry{
					URL: "invalid://url",
				},
				Name:    "mychart",
				Version: "1.0.0",
			},
			expectErr:   true,
			expectExist: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := c.chart.Pull()
			if (err != nil) != c.expectErr {
				t.Errorf("expected error: %v, got: %v", c.expectErr, err)
			}
			if p != "" && err != nil {
				t.Error("Path should be empty when err is returned")
			}
			b := fileExists(p)
			if b != c.expectExist {
				t.Errorf("expected tarPath does not exist: %v, got: %v", c.expectExist, b)
				os.RemoveAll(p)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	filename := "testfile"
	_, err := os.Create(filename)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer os.Remove(filename)

	if !fileExists(filename) {
		t.Errorf("expected file to exist")
	}

	if fileExists("nonexistentfile") {
		t.Errorf("expected file not to exist")
	}
}
