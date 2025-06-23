package external

import (
	"github.com/imposter-project/imposter-go/external/common"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegration_ExternalPluginLifecycle(t *testing.T) {
	pluginDir, _ := filepath.Abs("../bin")
	os.Setenv("IMPOSTER_PLUGIN_DIR", pluginDir)

	// Start plugins
	StartExternalPlugins()

	// Call handlers
	resp := InvokeExternalHandlers(common.HandlerRequest{Method: "get", Path: "/index.html"})
	assert.Contains(t, string(resp.Body), "<html", "Expected HTML response from Swagger UI plugin")

	resp2 := InvokeExternalHandlers(common.HandlerRequest{Method: "get", Path: "/does-not-exist"})
	assert.Equal(t, 404, resp2.StatusCode, "Expected 404 response for non-existent path")

	// Stop plugins
	StopExternalPlugins()
}
