package helm

import (
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/ChristofferNissen/helmper/pkg/util/file"
	"github.com/smallstep/assert"
	"github.com/stretchr/testify/mock"
	helm_registry "helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
)

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
			b := file.FileExists(p)
			if b != c.expectExist {
				t.Errorf("expected tarPath does not exist: %v, got: %v", c.expectExist, b)
				os.RemoveAll(p)
			}
		})
	}
}

// Define the mock registry client
type MockRegistryClient struct {
	mock.Mock
}

func (m *MockRegistryClient) Push(chart []byte, destination string, opts ...helm_registry.PushOption) (*helm_registry.PushResult, error) {
	args := m.Called(chart, destination)
	return to.Ptr(helm_registry.PushResult{
		Ref: destination,
	}), args.Error(1)
}

func (m *MockRegistryClient) Tags(ref string) ([]string, error) {
	args := m.Called(ref)
	return args.Get(0).([]string), args.Error(1)
}

func TestPush(t *testing.T) {
	// Create a mock registry client
	mockClient := new(MockRegistryClient)

	chart := Chart{
		Name:           "testchart",
		RegistryClient: mockClient,
	}

	chartFilePath := "/tmp/testchart.tgz"
	destination := "oci://localhost:5000/testchart:0.1.0"

	// Create a dummy chart file for testing
	err := os.WriteFile(chartFilePath, []byte("test data"), 0644)
	assert.NoError(t, err)
	defer os.Remove(chartFilePath)

	// Set up the expectations for the mock
	mockClient.On("Push", mock.Anything, destination).Return("success", nil)

	// Test the push function
	err = chart.push(chartFilePath, destination)
	assert.NoError(t, err)

	// Assert that the expectations were met
	mockClient.AssertExpectations(t)
}
