package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestYamlNodeToString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple string value",
			input:    "test string",
			expected: "test string",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "String with special characters",
			input:    "test-123_@#£",
			expected: "test-123_@#£",
		},
		{
			name:     "Numeric string",
			input:    "123",
			expected: "123",
		},
		{
			name:     "Multi-line string",
			input:    "line1\nline2\nline3",
			expected: "line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a YAML node with the test input
			node := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: tt.input,
			}

			result := yamlNodeToString(node)
			require.Equal(t, tt.expected, result)
		})
	}

	t.Run("Nil node", func(t *testing.T) {
		result := yamlNodeToString(nil)
		require.Empty(t, result)
	})

	t.Run("Non-string scalar", func(t *testing.T) {
		node := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!int",
			Value: "42",
		}
		result := yamlNodeToString(node)
		require.Equal(t, "42", result)
	})

	t.Run("Complex YAML structure", func(t *testing.T) {
		var node yaml.Node
		err := yaml.Unmarshal([]byte(`
key: value
list:
  - item1
  - item2
`), &node)
		require.NoError(t, err)
		result := yamlNodeToString(&node)
		require.Empty(t, result, "Complex YAML structure should not be converted to string")
	})
}
