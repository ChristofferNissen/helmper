package copa

import "github.com/aquasecurity/trivy/pkg/fanal/types"

func SupportedOS(os *types.OS) bool {
	if os == nil {
		return true
	}

	switch os.Family {
	case "photon":
		return false
	default:
		return true
	}
}
