package external

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIntegration_ExternalPluginLifecycle(t *testing.T) {
	// Set the IMPOSTER_PLUGIN_DIR to the swaggerui/impl directory
	pluginDir := filepath.Join("./swaggerui", "impl")
	os.Setenv("IMPOSTER_PLUGIN_DIR", pluginDir)

	// Start plugins
	StartExternalPlugins()

	// Call handlers
	InvokeExternalHandlers("/test-path")

	// Stop plugins
	StopExternalPlugins()
}
