package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
	"github.com/imposter-project/imposter-go/plugin"
	"github.com/stretchr/testify/require"
)

// TestCaptureDefaultStore_StoresRequestTemplate verifies that when no store is
// specified in the capture config, the captured value lands in the "request"
// store and can be retrieved via ${stores.request.<key>}.
func TestCaptureDefaultStore_StoresRequestTemplate(t *testing.T) {
	tempDir := t.TempDir()

	configContent := `plugin: rest
resources:
  - path: /users/{userName}
    method: POST
    capture:
      userName:
        pathParam: userName
    response:
      content: "Hello ${stores.request.userName}"
      template: true
      statusCode: 201`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins, err := plugin.LoadPlugins(configs, imposterConfig, nil)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(nil, w, r, plugins)
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/users/alice", "text/plain", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, 201, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Hello alice", string(body))
}

// TestContextRequestPathParams verifies ${context.request.pathParams.<key>}.
func TestContextRequestPathParams(t *testing.T) {
	tempDir := t.TempDir()

	configContent := `plugin: rest
resources:
  - path: /greeting/{name}
    method: GET
    response:
      content: "Hello ${context.request.pathParams.name}!"
      template: true
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins, err := plugin.LoadPlugins(configs, imposterConfig, nil)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(nil, w, r, plugins)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/greeting/world")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Hello world!", string(body))
}

// TestFallbackExpression verifies ${context.request.pathParams.nonexistent:-fallback}.
func TestFallbackExpression(t *testing.T) {
	tempDir := t.TempDir()

	configContent := `plugin: rest
resources:
  - path: /test
    method: GET
    response:
      content: "${context.request.pathParams.nonexistent:-fallback}"
      template: true
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins, err := plugin.LoadPlugins(configs, imposterConfig, nil)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(nil, w, r, plugins)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/test")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "fallback", string(body))
}

// TestContextRequestQueryParams verifies ${context.request.queryParams.<key>}.
func TestContextRequestQueryParams(t *testing.T) {
	tempDir := t.TempDir()

	configContent := `plugin: rest
resources:
  - path: /example
    method: GET
    response:
      content: "Hello ${context.request.queryParams.name}!"
      template: true
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins, err := plugin.LoadPlugins(configs, imposterConfig, nil)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(nil, w, r, plugins)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/example?name=World")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Hello World!", string(body))
}

// TestContextRequestHeaders verifies ${context.request.headers.<key>}.
func TestContextRequestHeaders(t *testing.T) {
	tempDir := t.TempDir()

	configContent := `plugin: rest
resources:
  - path: /headers
    method: GET
    response:
      content: "Header: ${context.request.headers.X-Custom}"
      template: true
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins, err := plugin.LoadPlugins(configs, imposterConfig, nil)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(nil, w, r, plugins)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/headers", nil)
	require.NoError(t, err)
	req.Header.Set("X-Custom", "test-value")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Header: test-value", string(body))
}

// TestContextRequestBody verifies ${context.request.body}.
func TestContextRequestBody(t *testing.T) {
	tempDir := t.TempDir()

	configContent := `plugin: rest
resources:
  - path: /body
    method: POST
    response:
      content: "Body: ${context.request.body}"
      template: true
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins, err := plugin.LoadPlugins(configs, imposterConfig, nil)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(nil, w, r, plugins)
	}))
	defer server.Close()

	resp, err := http.Post(server.URL+"/body", "text/plain", strings.NewReader("hello"))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, 200, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Body: hello", string(body))
}
