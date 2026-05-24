package test

import (
	"fmt"
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

// startPassthroughServer loads the given config content and starts an Imposter
// test server in front of it.
func startPassthroughServer(t *testing.T, configContent string) *httptest.Server {
	t.Helper()
	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644))

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins, err := plugin.LoadPlugins(configs, imposterConfig, nil)
	require.NoError(t, err)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(nil, w, r, plugins)
	}))
}

func TestIntegration_Passthrough_HappyPath(t *testing.T) {
	var gotPath, gotQuery, gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("X-Upstream", "yes")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream body"))
	}))
	defer upstream.Close()

	cfg := fmt.Sprintf(`plugin: rest
upstreams:
  backend:
    url: %s/base
resources:
  - path: /api/users
    method: POST
    passthrough: backend`, upstream.URL)

	server := startPassthroughServer(t, cfg)
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/users?page=2", "application/json", strings.NewReader(`{"k":"v"}`))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "upstream body", string(body))
	require.Equal(t, "yes", resp.Header.Get("X-Upstream"))
	require.Equal(t, "/base/api/users", gotPath)
	require.Equal(t, "page=2", gotQuery)
	require.Equal(t, `{"k":"v"}`, gotBody)
}

func TestIntegration_Passthrough_UpstreamErrorVerbatim(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream error"))
	}))
	defer upstream.Close()

	cfg := fmt.Sprintf(`plugin: rest
upstreams:
  backend:
    url: %s
resources:
  - path: /thing
    method: GET
    passthrough: backend`, upstream.URL)

	server := startPassthroughServer(t, cfg)
	defer server.Close()

	resp, err := http.Get(server.URL + "/thing")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	require.Equal(t, "upstream error", string(body))
}

func TestIntegration_Passthrough_UpstreamDownReturns502(t *testing.T) {
	cfg := `plugin: rest
upstreams:
  backend:
    url: http://127.0.0.1:1
resources:
  - path: /thing
    method: GET
    passthrough: backend`

	server := startPassthroughServer(t, cfg)
	defer server.Close()

	resp, err := http.Get(server.URL + "/thing")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadGateway, resp.StatusCode)
}

func TestIntegration_Passthrough_UnknownUpstreamFailsToLoad(t *testing.T) {
	tempDir := t.TempDir()
	cfg := `plugin: rest
upstreams:
  backend:
    url: http://api.example.com
resources:
  - path: /thing
    method: GET
    passthrough: missing`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(cfg), 0644))

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	_, err := plugin.LoadPlugins(configs, imposterConfig, nil)
	require.Error(t, err)
}
