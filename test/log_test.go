package test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/imposter-project/imposter-go/plugin"
	"github.com/imposter-project/imposter-go/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceLog(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create test configuration with log feature
	configContent := `plugin: rest
resources:
  - path: /example/{id}
    method: GET
    log: "Received request for ID: ${context.request.pathParams.id} from ${context.request.headers.X-Client-ID}"
    response:
      content: "Response for ${context.request.pathParams.id}"
      statusCode: 200
      template: true`

	err := os.WriteFile(tempDir+"/test-config.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	imposterConfig := &config.ImposterConfig{
		ServerPort: "8080",
		ServerUrl:  "http://localhost:8080",
	}
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins := plugin.LoadPlugins(configs, tempDir, imposterConfig)

	// Capture log output
	var logOutput bytes.Buffer
	originalOutput, originalError := logger.GetSinks()
	logger.SetOutputSink(&logOutput)
	logger.SetErrorSink(&logOutput)
	defer func() {
		logger.SetOutputSink(originalOutput)
		logger.SetErrorSink(originalError)
	}()

	// Test 1: Simple path parameter test
	req, err := http.NewRequest("GET", "/example/123", strings.NewReader(""))
	require.NoError(t, err)
	req.Header.Set("X-Client-ID", "test-client")

	rec := httptest.NewRecorder()
	handler.HandleRequest(imposterConfig, rec, req, plugins)

	// Check response
	assert.Equal(t, http.StatusOK, rec.Code)
	responseBody, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	assert.Equal(t, "Response for 123", string(responseBody))

	// Check log output - it should contain our log template message
	logString := logOutput.String()
	assert.Contains(t, logString, "Received request for ID: 123 from test-client")

	// Reset log buffer for next test
	logOutput.Reset()

	// Test 2: Different client and path parameter
	req, err = http.NewRequest("GET", "/example/456", nil)
	require.NoError(t, err)
	req.Header.Set("X-Client-ID", "mobile-app")

	rec = httptest.NewRecorder()
	handler.HandleRequest(imposterConfig, rec, req, plugins)

	// Check response
	assert.Equal(t, http.StatusOK, rec.Code)
	responseBody, err = io.ReadAll(rec.Body)
	require.NoError(t, err)
	assert.Equal(t, "Response for 456", string(responseBody))

	// Check log output - it should contain our log template message
	logString = logOutput.String()
	assert.Contains(t, logString, "Received request for ID: 456 from mobile-app")
}

func TestInterceptorLog(t *testing.T) {
	// Test interceptors with log feature
	matcher := config.MatchCondition{
		Value:    "Some-User-Agent",
		Operator: "EqualTo",
	}

	// Create an interceptor with a log message - set continue to false to ensure it sets the resource
	interceptor := testutils.NewInterceptor("GET", "/secured", map[string]config.MatcherUnmarshaler{
		"User-Agent": {Matcher: matcher},
	}, &config.Response{
		StatusCode: 200,
		Content:    "Authenticated",
	}, false)
	
	// Add the log message to the interceptor
	interceptor.BaseResource.Log = "Authenticated request from agent: ${context.request.headers.User-Agent}"

	configs := []config.Config{
		{
			Plugin: "rest",
			Resources: []config.Resource{
				testutils.NewResource("GET", "/secured", &config.Response{
					StatusCode: 200,
					Content:    "Protected content",
				}),
			},
			Interceptors: []config.Interceptor{interceptor},
		},
	}

	imposterConfig := &config.ImposterConfig{
		ServerPort: "8080",
	}
	plugins := plugin.LoadPlugins(configs, "", imposterConfig)

	// Capture log output
	var logOutput bytes.Buffer
	originalOutput, originalError := logger.GetSinks()
	logger.SetOutputSink(&logOutput)
	logger.SetErrorSink(&logOutput)
	defer func() {
		logger.SetOutputSink(originalOutput)
		logger.SetErrorSink(originalError)
	}()

	// Test with matching user agent
	req, err := http.NewRequest("GET", "/secured", new(strings.Reader))
	require.NoError(t, err)
	req.Header.Set("User-Agent", "Some-User-Agent")

	rec := httptest.NewRecorder()
	handler.HandleRequest(imposterConfig, rec, req, plugins)

	// Check response
	assert.Equal(t, http.StatusOK, rec.Code)

	// Check log output - it should contain our log template message
	logString := logOutput.String()
	assert.Contains(t, logString, "Authenticated request from agent: Some-User-Agent")
}

func TestComplexLogTemplates(t *testing.T) {
	// Test complex template combinations in log messages
	
	// Create resources with complex log templates
	resource := testutils.NewResource("POST", "/data", &config.Response{
		StatusCode: 201,
		Content:    "Created",
	})
	
	// Add a log template with various template functions
	resource.BaseResource.Log = "Request: method=${context.request.method}, " +
		"timestamp=${datetime.now.iso8601_datetime}, " +
		"id=${random.uuid()}, " +
		"client=${context.request.headers.X-Client-ID:-unknown}"

	configs := []config.Config{
		{
			Plugin:    "rest",
			Resources: []config.Resource{resource},
		},
	}

	imposterConfig := &config.ImposterConfig{
		ServerPort: "8080",
	}
	plugins := plugin.LoadPlugins(configs, "", imposterConfig)

	// Capture log output
	var logOutput bytes.Buffer
	originalOutput, originalError := logger.GetSinks()
	logger.SetOutputSink(&logOutput)
	logger.SetErrorSink(&logOutput)
	defer func() {
		logger.SetOutputSink(originalOutput)
		logger.SetErrorSink(originalError)
	}()

	// Send request
	req, err := http.NewRequest("POST", "/data", strings.NewReader("test data"))
	require.NoError(t, err)
	
	// Test 1: With client ID header
	req.Header.Set("X-Client-ID", "test-app")
	
	rec := httptest.NewRecorder()
	handler.HandleRequest(imposterConfig, rec, req, plugins)

	// Check response
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Check log output - it should contain all templated parts
	logString := logOutput.String()
	assert.Contains(t, logString, "Request: method=POST")
	assert.Contains(t, logString, "timestamp=")  // We can't check the exact timestamp
	assert.Contains(t, logString, "id=")         // We can't check the exact UUID
	assert.Contains(t, logString, "client=test-app")
	
	// Reset log buffer for next test
	logOutput.Reset()
	
	// Test 2: Without client ID header (should use default)
	req.Header.Del("X-Client-ID")
	
	rec = httptest.NewRecorder()
	handler.HandleRequest(imposterConfig, rec, req, plugins)

	// Check log output again
	logString = logOutput.String()
	assert.Contains(t, logString, "client=unknown")
}