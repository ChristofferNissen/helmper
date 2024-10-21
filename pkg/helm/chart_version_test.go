package helm

import (
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

type MockIndexFileLoader struct {
	mock.Mock
}

func (m *MockIndexFileLoader) LoadIndexFile(indexFilePath string) (*repo.IndexFile, error) {
	args := m.Called(indexFilePath)
	return args.Get(0).(*repo.IndexFile), args.Error(1)
}

func TestVersionsInRange(t *testing.T) {
	mockRepoIndex := &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			"testchart": {
				{
					Metadata: &chart.Metadata{
						Name:    "testchart",
						Version: "1.0.0",
					},
				},
				{
					Metadata: &chart.Metadata{
						Name:    "testchart",
						Version: "1.1.0",
					},
				},
			},
		},
	}

	mockLoader := new(MockIndexFileLoader)
	mockLoader.On("LoadIndexFile", mock.Anything).Return(mockRepoIndex, nil)

	r, _ := semver.ParseRange(">= 1.0.0")
	c := Chart{
		Repo:            repo.Entry{Name: "testrepo"},
		Name:            "testchart",
		Version:         "1.0.0",
		IndexFileLoader: mockLoader,
	}

	versions, err := VersionsInRange(r, c)
	assert.NoError(t, err)
	assert.Equal(t, []string{"1.1.0", "1.0.0"}, versions)

	mockLoader.AssertExpectations(t)
}

func TestResolveVersions(t *testing.T) {
	mockClient := new(MockRegistryClient)

	c := Chart{
		Repo: repo.Entry{
			URL: "oci://localhost:5000/testchart",
		},
		Name:           "testchart",
		Version:        ">= 1.0.0",
		PlainHTTP:      true,
		RegistryClient: mockClient,
	}

	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0"}, nil)

	versions, err := c.ResolveVersions()
	assert.NoError(t, err)
	assert.Equal(t, []string{"1.0.0", "1.1.0"}, versions)
}

func TestResolveVersion(t *testing.T) {
	mockClient := new(MockRegistryClient)

	c := Chart{
		Repo: repo.Entry{
			URL: "oci://localhost:5000/testchart",
		},
		Name:           "testchart",
		Version:        ">= 1.0.0",
		PlainHTTP:      true,
		RegistryClient: mockClient,
	}

	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0"}, nil)

	version, err := c.ResolveVersion()
	assert.NoError(t, err)
	assert.Equal(t, "1.1.0", version)
}

func TestLatestVersion(t *testing.T) {
	mockClient := new(MockRegistryClient)

	c := Chart{
		Repo: repo.Entry{
			URL: "oci://localhost:5000/testchart",
		},
		Name:           "testchart",
		Version:        ">= 1.0.0",
		PlainHTTP:      true,
		RegistryClient: mockClient,
	}

	mockClient.On("Tags", mock.Anything).Return([]string{"1.0.0", "1.1.0"}, nil)

	version, err := c.LatestVersion()
	assert.NoError(t, err)
	assert.Equal(t, "1.1.0", version)
}

// func TestResolveVersions2(t *testing.T) {

// 	c := ChartCollection{
// 		Charts: []*Chart{
// 			{
// 				Name:    "argo-cd",
// 				Version: ">4.0.0 <5.0.0",
// 				Repo: repo.Entry{
// 					Name: "argoproj",
// 					URL:  "https://argoproj.github.io/argo-helm",
// 				},
// 			},
// 		},
// 	}

// 	co := ChartOption{
// 		ChartCollection: &c,
// 	}
// 	_, err := co.ChartCollection.SetupHelm()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	vs, err := c.Charts[0].ResolveVersions()
// 	if err != nil {
// 		t.Errorf("want '%s' goChartt '%s'", "nil", err.Error())
// 	}

// 	if len(vs) != 63 {
// 		t.Errorf("want '%s' got '%d'", "63", len(vs))
// 	}
// }

// func TestResolveVersions3(t *testing.T) {

// 	c := ChartCollection{
// 		Charts: []*Chart{
// 			{
// 				Name:    "argo-cd",
// 				Version: ">5.51.0 <6.0.0",
// 				Repo: repo.Entry{
// 					Name: "argoproj",
// 					URL:  "https://argoproj.github.io/argo-helm",
// 				},
// 			},
// 		},
// 	}

// 	co := ChartOption{
// 		ChartCollection: &c,
// 	}
// 	_, err := co.ChartCollection.SetupHelm()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	vs, err := c.Charts[0].ResolveVersions()
// 	if err != nil {
// 		t.Errorf("want '%s' got '%s'", "err", "nil")
// 	}

// 	if len(vs) != 26 {
// 		t.Errorf("want '%s' got '%d'", "26", len(vs))
// 	}
// }
