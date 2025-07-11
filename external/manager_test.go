package external

import (
	"github.com/imposter-project/imposter-go/external/shared"
	"github.com/imposter-project/imposter-go/plugin"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPluginFilePath(t *testing.T) {
	originalPluginOs := pluginOs
	defer func() {
		pluginOs = originalPluginOs
	}()

	originalPluginDir := pluginDir
	defer func() {
		pluginDir = originalPluginDir
	}()

	pluginDir = "/test/plugins"

	tests := []struct {
		name       string
		pluginName string
		os         string
		expected   string
	}{
		{
			name:       "Linux plugin path",
			pluginName: "swagger",
			os:         "linux",
			expected:   "/test/plugins/plugin-swagger",
		},
		{
			name:       "Windows plugin path",
			pluginName: "swagger",
			os:         "windows",
			expected:   "/test/plugins/plugin-swagger.exe",
		},
		{
			name:       "Darwin plugin path",
			pluginName: "openapi",
			os:         "darwin",
			expected:   "/test/plugins/plugin-openapi",
		},
		{
			name:       "Windows plugin path with complex name",
			pluginName: "my-complex-plugin",
			os:         "windows",
			expected:   "/test/plugins/plugin-my-complex-plugin.exe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pluginOs = tt.os
			result := getPluginFilePath(tt.pluginName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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

func TestIntegration_ExternalPluginLifecycle(t *testing.T) {
	pluginDir, _ := filepath.Abs("../bin")

	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		t.Skipf("Plugin directory %s does not exist, skipping test", pluginDir)
	}

	_ = os.Setenv("IMPOSTER_PLUGIN_DIR", pluginDir)
	_ = os.Setenv("IMPOSTER_EXTERNAL_PLUGINS", "true")

	var plugins []plugin.Plugin

	// Start plugins
	err := StartExternalPlugins(plugins)
	require.NoError(t, err)

	// Call handlers
	resp := InvokeExternalHandlers(shared.HandlerRequest{Method: "get", Path: "/_spec/"})
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode, "Expected 200 response for index")
	assert.Contains(t, string(resp.Body), "<html", "Expected HTML response from Swagger UI plugin")

	resp2 := InvokeExternalHandlers(shared.HandlerRequest{Method: "get", Path: "/does-not-exist"})
	require.NotNil(t, resp2)
	assert.Equal(t, 404, resp2.StatusCode, "Expected 404 response for non-existent path")

	resp3 := InvokeExternalHandlers(shared.HandlerRequest{Method: "post", Path: "/index.html"})
	require.NotNil(t, resp3)
	assert.Equal(t, 405, resp3.StatusCode, "Expected 405 response for unsupported method")

	// Stop plugins
	StopExternalPlugins()
}
