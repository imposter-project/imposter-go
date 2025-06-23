package external

import (
	"github.com/imposter-project/imposter-go/external/common"
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
	InvokeExternalHandlers(common.HandlerArgs{Method: "get", Path: "/index.html"})

	// Stop plugins
	StopExternalPlugins()
}
