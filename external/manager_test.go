package external

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPluginNameFromFileName(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		expected string
	}{
		{
			name:     "Linux plugin file",
			fileName: "plugin-swagger",
			expected: "swagger",
		},
		{
			name:     "Windows plugin file",
			fileName: "plugin-swagger.exe",
			expected: "swagger",
		},
		{
			name:     "Complex plugin name Linux",
			fileName: "plugin-my-complex-plugin",
			expected: "my-complex-plugin",
		},
		{
			name:     "Complex plugin name Windows",
			fileName: "plugin-my-complex-plugin.exe",
			expected: "my-complex-plugin",
		},
		{
			name:     "Plugin with dashes",
			fileName: "plugin-some-long-name",
			expected: "some-long-name",
		},
		{
			name:     "Plugin with dashes Windows",
			fileName: "plugin-some-long-name.exe",
			expected: "some-long-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPluginNameFromFileName(tt.fileName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
