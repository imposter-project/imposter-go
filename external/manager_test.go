package external

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIntegration_ExternalPluginLifecycle(t *testing.T) {
	pluginDir, _ := filepath.Abs("../bin")
	os.Setenv("IMPOSTER_PLUGIN_DIR", pluginDir)

	// Start plugins
	StartExternalPlugins()

	// Call handlers
	InvokeExternalHandlers("/test-path")

	// Stop plugins
	StopExternalPlugins()
}
