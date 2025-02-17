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
	"github.com/imposter-project/imposter-go/test/testutils"
	"github.com/stretchr/testify/require"
)

func TestHomeRoute(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create test configuration
	configContent := `plugin: rest
resources:
  - path: /
    method: GET
    response:
      content: "Hello, world!"
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins := plugin.LoadPlugins(configs, tempDir, imposterConfig)

	// Start the server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(nil, w, r, plugins)
	}))
	defer server.Close()

	// Make request
	resp, err := http.Get(server.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check response
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Hello, world!", string(body))
}

func TestIntegration_MatchJSONBody(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create test configuration
	configContent := `plugin: rest
resources:
  - path: /example
    method: POST
    requestBody:
      value: '{"name": "test"}'
    response:
      content: "Matched JSON body"
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins := plugin.LoadPlugins(configs, tempDir, imposterConfig)

	// Start the server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(nil, w, r, plugins)
	}))
	defer server.Close()

	// Make request with matching JSON body
	jsonBody := `{"name": "test"}`
	resp, err := http.Post(server.URL+"/example", "application/json", strings.NewReader(jsonBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check response
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "Matched JSON body", string(body))

	// Make request with non-matching JSON body
	jsonBody = `{"name": "wrong"}`
	resp, err = http.Post(server.URL+"/example", "application/json", strings.NewReader(jsonBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check response
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestInterceptors_ShortCircuit(t *testing.T) {
	matcher := config.MatchCondition{
		Value:    "Some-User-Agent",
		Operator: "NotEqualTo",
	}

	configs := []config.Config{
		{
			Plugin: "rest",
			Resources: []config.Resource{
				testutils.NewResource("GET", "/example", config.Response{
					StatusCode: 200,
					Content:    "Hello, world!",
				}),
			},
			Interceptors: []config.Interceptor{
				testutils.NewInterceptor("GET", "/example", map[string]config.MatcherUnmarshaler{
					"User-Agent": {Matcher: matcher},
				}, &config.Response{
					StatusCode: 400,
					Content:    "Invalid user agent",
				}, false),
			},
		},
	}
	imposterConfig := &config.ImposterConfig{
		ServerPort: "8080",
	}
	plugins := plugin.LoadPlugins(configs, "", imposterConfig)

	// Test with invalid user agent
	req, err := http.NewRequest("GET", "/example", new(strings.Reader))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("User-Agent", "Invalid-Agent")

	rec := httptest.NewRecorder()

	handler.HandleRequest(imposterConfig, rec, req, plugins)

	if status := rec.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, status)
	}

	expectedBody := "Invalid user agent"
	if rec.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, rec.Body.String())
	}

	// Test with valid user agent
	req.Header.Set("User-Agent", "Some-User-Agent")
	rec = httptest.NewRecorder()
	handler.HandleRequest(imposterConfig, rec, req, plugins)

	if status := rec.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	expectedBody = "Hello, world!"
	if rec.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, rec.Body.String())
	}
}

func TestInterceptors_Passthrough(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }
	in := testutils.NewInterceptorWithResponse("GET", "/example", true)
	in.Capture = map[string]config.Capture{
		"userAgent": {
			Enabled: boolPtr(true),
			Store:   "request",
			CaptureConfig: config.CaptureConfig{
				RequestHeader: "User-Agent",
			},
		},
	}
	configs := []config.Config{
		{
			Plugin: "rest",
			Resources: []config.Resource{
				testutils.NewResource("GET", "/example", config.Response{
					StatusCode: 200,
					Content:    "User agent: ${stores.request.userAgent}",
					Template:   true,
				}),
			},
			Interceptors: []config.Interceptor{in},
		},
	}
	imposterConfig := &config.ImposterConfig{
		ServerPort: "8080",
	}
	plugins := plugin.LoadPlugins(configs, "", imposterConfig)

	req, err := http.NewRequest("GET", "/example", new(strings.Reader))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("User-Agent", "Test-Agent")

	rec := httptest.NewRecorder()
	handler.HandleRequest(&config.ImposterConfig{}, rec, req, plugins)

	if status := rec.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	expectedBody := "User agent: Test-Agent"
	if rec.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, rec.Body.String())
	}
}
