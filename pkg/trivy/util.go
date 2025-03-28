package trivy

import (
	"github.com/aquasecurity/trivy/pkg/types"
)

func ContainsOsPkgs(rs types.Results) bool {
	for _, r := range rs {
		switch r.Class {
		case types.ClassOSPkg:
			if !r.IsEmpty() {
				return true
			}
		}
	}
	return false
}
