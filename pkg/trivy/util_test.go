package trivy

import (
	"testing"

	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestContainsOsPkgs(t *testing.T) {
	// Test case where result contains OS packages
	resultsWithOSPkg := types.Results{
		{
			Class: types.ClassOSPkg,
			Vulnerabilities: []types.DetectedVulnerability{
				{VulnerabilityID: "CVE-2021-1234"},
			},
		},
	}
	assert.True(t, ContainsOsPkgs(resultsWithOSPkg))

	// Test case where result does not contain OS packages
	resultsWithoutOSPkg := types.Results{
		{
			Class: "SomeOtherClass",
			Vulnerabilities: []types.DetectedVulnerability{
				{VulnerabilityID: "CVE-2021-1234"},
			},
		},
	}
	assert.False(t, ContainsOsPkgs(resultsWithoutOSPkg))

	// Test case where result contains OS packages but they are empty
	resultsWithEmptyOSPkg := types.Results{
		{
			Class: types.ClassOSPkg,
		},
	}
	assert.False(t, ContainsOsPkgs(resultsWithEmptyOSPkg))

	// Test case where results are empty
	emptyResults := types.Results{}
	assert.False(t, ContainsOsPkgs(emptyResults))
}
