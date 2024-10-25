package helm

import "testing"

func TestConditionMet(t *testing.T) {
	type input struct {
		condition      string
		values         map[string]any
		expectedResult bool
	}

	tests := []input{
		{
			condition: "test.enabled",
			values: map[string]any{
				"test": map[string]any{
					"enabled": true,
				},
			},
			expectedResult: true,
		},
		{
			condition: "test.enabled",
			values: map[string]any{
				"test": map[string]any{
					"enabled": false,
				},
			},
			expectedResult: false,
		},
		{
			condition: "service.enabled",
			values: map[string]any{
				"test": map[string]any{
					"enabled": true,
				},
			},
			expectedResult: false,
		},
		{
			condition: "service.enabled",
			values: map[string]any{
				"test": map[string]any{
					"enabled": true,
				},
				"service": map[string]any{
					"enabled": true,
				},
			},
			expectedResult: true,
		},
		{
			condition: "service.enabled",
			values: map[string]any{
				"test": map[string]any{
					"enabled": true,
				},
				"other": map[string]any{
					"service": map[string]any{
						"enabled": true,
					},
				},
			},
			expectedResult: false,
		},
	}

	for _, test := range tests {
		res := ConditionMet(test.condition, test.values)
		if res != test.expectedResult {
			t.Errorf("got '%t' want '%t'", res, test.expectedResult)
		}
	}
}
