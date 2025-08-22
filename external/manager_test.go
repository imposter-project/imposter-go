package external

import (
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
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

func TestListRequestedPlugins(t *testing.T) {
	tests := []struct {
		name     string
		configs  []config.Config
		expected []string
	}{
		{
			name:     "Empty configs",
			configs:  []config.Config{},
			expected: nil,
		},
		{
			name: "Single config",
			configs: []config.Config{
				{Plugin: "rest"},
			},
			expected: []string{"rest"},
		},
		{
			name: "Multiple configs with unique plugins",
			configs: []config.Config{
				{Plugin: "rest"},
				{Plugin: "openapi"},
				{Plugin: "soap"},
			},
			expected: []string{"rest", "openapi", "soap"},
		},
		{
			name: "Multiple configs with duplicate plugins",
			configs: []config.Config{
				{Plugin: "rest"},
				{Plugin: "openapi"},
				{Plugin: "rest"},
				{Plugin: "soap"},
				{Plugin: "openapi"},
			},
			expected: []string{"rest", "openapi", "soap"},
		},
		{
			name: "All same plugin",
			configs: []config.Config{
				{Plugin: "rest"},
				{Plugin: "rest"},
				{Plugin: "rest"},
			},
			expected: []string{"rest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := listRequestedPlugins(tt.configs)
			assert.Equal(t, tt.expected, result)
		})
	}
}
