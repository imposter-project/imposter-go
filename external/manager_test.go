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

func TestIntegration_ExternalPluginLifecycle(t *testing.T) {
	pluginDir, _ := filepath.Abs("../bin")
	os.Setenv("IMPOSTER_PLUGIN_DIR", pluginDir)

	var configs []config.Config

	// Start plugins
	err := StartExternalPlugins(configs)
	require.NoError(t, err)

	// Call handlers
	resp := InvokeExternalHandlers(handler.HandlerRequest{Method: "get", Path: "/_spec/index.html"})
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode, "Expected 200 response for index.html")
	assert.Contains(t, string(resp.Body), "<html", "Expected HTML response from Swagger UI plugin")

	resp = InvokeExternalHandlers(handler.HandlerRequest{Method: "get", Path: "/_spec/"})
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode, "Expected 200 response for /")
	assert.Contains(t, string(resp.Body), "<html", "Expected HTML response from Swagger UI plugin")

	resp2 := InvokeExternalHandlers(handler.HandlerRequest{Method: "get", Path: "/does-not-exist"})
	require.NotNil(t, resp2)
	assert.Equal(t, 404, resp2.StatusCode, "Expected 404 response for non-existent path")

	resp3 := InvokeExternalHandlers(handler.HandlerRequest{Method: "post", Path: "/index.html"})
	require.NotNil(t, resp3)
	assert.Equal(t, 405, resp3.StatusCode, "Expected 405 response for unsupported method")

	// Stop plugins
	StopExternalPlugins()
}
