package helm

import (
	"fmt"
	"strings"

	"github.com/ChristofferNissen/helmper/helmper/pkg/registry"
	"github.com/ChristofferNissen/helmper/helmper/pkg/util/ternary"
)

// traverse helm chart values to determine if condition is met
func ConditionMet(condition string, values map[string]any) bool {
	pos := values
	enabled := false
	for _, e := range strings.Split(condition, ".") {
		switch v := pos[e].(type) {
		case string:
			enabled = v == "true"
		case bool:
			enabled = v
		case map[string](any):
			pos = v
		case interface{}:
			pos = pos[e].(map[string]any)
		}
	}
	return enabled
}

// traverse helm chart values data structure
func findImageReferencesAcc(data map[string]any, values map[string]any, acc string) map[*registry.Image][]string {
	res := make(map[*registry.Image][]string)

	i := registry.Image{}
	for k, v := range data {
		switch v := v.(type) {
		// yaml value
		case string:
			found := true

			switch k {
			case "registry":
				i.Registry = v
			case "repository":
				i.Repository = v
			case "image":
				i.Repository = v
			case "tag":
				i.Tag = v
			case "digest":
				i.Digest = v
			case "sha":
				i.Digest = v
			default:
				found = false
			}

			if found {
				res[&i] = append(res[&i], fmt.Sprintf("%s.%s", acc, k))
			}

		// nested yaml object
		case map[string]any:
			// same path in yaml

			// Only parsed enabled sections
			enabled := true
			for k1, v1 := range v {
				if k1 == "enabled" {
					switch value := v1.(type) {
					case string:
						enabled = value == "true"
					case bool:
						enabled = ConditionMet(k1, values[k].(map[string]any))
					}
				}
			}

			// if enabled, parse nested section
			if enabled {
				path := ternary.Ternary(acc == "", k, fmt.Sprintf("%s.%s", acc, k))
				nestedRes := findImageReferencesAcc(v, values[k].(map[string]any), path)
				for k, v := range nestedRes {
					res[k] = v
				}
			}
		}
	}

	return res
}

func findImageReferences(data map[string]any, values map[string]any) map[*registry.Image][]string {
	return findImageReferencesAcc(data, values, "")
}
