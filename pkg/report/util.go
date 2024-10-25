package report

import "github.com/ChristofferNissen/helmper/pkg/util/file"

func DeterminePathType(path string) string {
	// Output Table
	if file.Exists(path) {
		return "custom"
	}
	return "default"
}
