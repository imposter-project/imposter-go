package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonPathQuery(t *testing.T) {
	tests := []struct {
		name          string
		json          []byte
		jsonPathExpr  string
		expectedValue interface{}
		expectSuccess bool
	}{
		{
			name:          "simple string value",
			json:          []byte(`{"name": "test"}`),
			jsonPathExpr:  "$.name",
			expectedValue: "test",
			expectSuccess: true,
		},
		{
			name:          "nested object value",
			json:          []byte(`{"person": {"name": "John", "age": 30}}`),
			jsonPathExpr:  "$.person.age",
			expectedValue: float64(30),
			expectSuccess: true,
		},
		{
			name:          "array element",
			json:          []byte(`{"items": ["one", "two", "three"]}`),
			jsonPathExpr:  "$.items[1]",
			expectedValue: "two",
			expectSuccess: true,
		},
		{
			name:          "invalid JSON",
			json:          []byte(`{"invalid json`),
			jsonPathExpr:  "$.name",
			expectedValue: nil,
			expectSuccess: false,
		},
		{
			name:          "invalid JSONPath expression",
			json:          []byte(`{"name": "test"}`),
			jsonPathExpr:  "$.[invalid",
			expectedValue: nil,
			expectSuccess: false,
		},
		{
			name:          "non-existent path",
			json:          []byte(`{"name": "test"}`),
			jsonPathExpr:  "$.age",
			expectedValue: nil,
			expectSuccess: false,
		},
		{
			name:          "complex nested array",
			json:          []byte(`{"users": [{"name": "John", "scores": [85, 90, 95]}, {"name": "Jane", "scores": [88, 92, 98]}]}`),
			jsonPathExpr:  "$.users[1].scores[2]",
			expectedValue: float64(98),
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, success := JsonPathQuery(tt.json, tt.jsonPathExpr)
			assert.Equal(t, tt.expectSuccess, success)
			if tt.expectSuccess {
				assert.Equal(t, tt.expectedValue, result)
			}
		})
	}
}
