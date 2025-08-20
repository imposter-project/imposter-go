package external

import (
	"github.com/imposter-project/imposter-go/external/shared"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
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

func TestIntegration_ExternalPluginLifecycle(t *testing.T) {
	pluginDir, _ := filepath.Abs("../bin")

	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		t.Skipf("Plugin directory %s does not exist, skipping test", pluginDir)
	}

	_ = os.Setenv("IMPOSTER_PLUGIN_DIR", pluginDir)
	_ = os.Setenv("IMPOSTER_EXTERNAL_PLUGINS", "true")

	var configs []config.Config

	// Start plugins
	imposterConfig := &config.ImposterConfig{
		ServerUrl: "http://localhost:8080",
	}
	loaded, err := StartExternalPlugins(imposterConfig, configs)
	require.NoError(t, err)
	require.NotEmpty(t, loaded, "Expected external plugins to be loaded")

	// Check if plugins are loaded
	require.NotEmpty(t, loaded, "Expected plugins to be loaded")

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
