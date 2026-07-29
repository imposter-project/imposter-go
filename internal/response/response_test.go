package response

import (
	"github.com/imposter-project/imposter-go/internal/exchange"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
)

const delayTolerance = 100 // milliseconds

func TestNewResponseState(t *testing.T) {
	rs := NewResponseState()
	assert.Equal(t, http.StatusOK, rs.StatusCode)
	assert.NotNil(t, rs.Headers)
	assert.Empty(t, rs.Headers)
	assert.Nil(t, rs.Body)
	assert.False(t, rs.Stopped)
	assert.False(t, rs.Handled)
}

func TestWriteToResponseWriter(t *testing.T) {
	tests := []struct {
		name           string
		responseState  *exchange.ResponseState
		expectedStatus int
		expectedBody   string
		expectedHeader map[string]string
	}{
		{
			name: "normal response",
			responseState: &exchange.ResponseState{
				StatusCode: http.StatusOK,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       []byte(`{"status":"ok"}`),
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
			expectedHeader: map[string]string{"Content-Type": "application/json"},
		},
		{
			name: "stopped response",
			responseState: &exchange.ResponseState{
				StatusCode: http.StatusOK,
				Stopped:    true,
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "HTTP server does not support connection hijacking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.responseState.WriteToResponseWriter(w)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
			for k, v := range tt.expectedHeader {
				assert.Equal(t, v, w.Header().Get(k))
			}
		})
	}
}

func TestSimulateDelay(t *testing.T) {
	tests := []struct {
		name          string
		delay         config.Delay
		expectedDelay time.Duration
	}{
		{
			name:          "exact delay",
			delay:         config.Delay{Exact: 100},
			expectedDelay: 100 * time.Millisecond,
		},
		{
			name:          "range delay",
			delay:         config.Delay{Min: 50, Max: 150},
			expectedDelay: 50 * time.Millisecond, // We'll verify it's at least the minimum
		},
		{
			name:          "no delay",
			delay:         config.Delay{},
			expectedDelay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			start := time.Now()
			SimulateDelay(tt.delay, req)
			elapsed := time.Since(start)

			if tt.delay.Exact > 0 {
				assert.InDelta(t, tt.expectedDelay, elapsed, float64(delayTolerance*time.Millisecond))
			} else if tt.delay.Min > 0 && tt.delay.Max > 0 {
				assert.GreaterOrEqual(t, elapsed, tt.expectedDelay)
				assert.LessOrEqual(t, elapsed, time.Duration(tt.delay.Max+delayTolerance)*time.Millisecond)
			} else {
				assert.Less(t, elapsed, delayTolerance*time.Millisecond)
			}
		})
	}
}

func TestSimulateFailure(t *testing.T) {
	tests := []struct {
		name        string
		failureType string
		expectStop  bool
		expectNil   bool
	}{
		{
			name:        "empty response",
			failureType: "EmptyResponse",
			expectStop:  false,
			expectNil:   true,
		},
		{
			name:        "close connection",
			failureType: "CloseConnection",
			expectStop:  true,
			expectNil:   false,
		},
		{
			name:        "unknown failure",
			failureType: "Unknown",
			expectStop:  false,
			expectNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := NewResponseState()
			rs.Body = []byte("test")
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			handled := SimulateFailure(rs, tt.failureType, req)

			if tt.failureType == "Unknown" {
				assert.False(t, handled)
				return
			}

			assert.True(t, handled)
			assert.Equal(t, tt.expectStop, rs.Stopped)
			if tt.expectNil {
				assert.Nil(t, rs.Body)
			}
		})
	}
}

func TestProcessResponse(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	testFilePath := "test.txt"
	testFileContent := "test file content"
	err := os.WriteFile(tmpDir+"/"+testFilePath, []byte(testFileContent), 0644)
	assert.NoError(t, err)

	// Create a test directory with response files for directory-based tests
	testDirPath := "responses"
	err = os.Mkdir(tmpDir+"/"+testDirPath, 0755)
	assert.NoError(t, err)
	err = os.WriteFile(tmpDir+"/"+testDirPath+"/index.html", []byte("index file content"), 0644)
	assert.NoError(t, err)

	// Create a subdirectory with an index.html file
	err = os.MkdirAll(tmpDir+"/"+testDirPath+"/subdir", 0755)
	assert.NoError(t, err)
	err = os.WriteFile(tmpDir+"/"+testDirPath+"/subdir/specific.json", []byte(`{"specific":"response"}`), 0644)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		response       config.Response
		expectedStatus int
		expectedBody   string
		expectedHeader map[string]string
		requestPath    string
		requestMatcher *config.RequestMatcher
	}{
		{
			name: "basic response",
			response: config.Response{
				StatusCode: http.StatusCreated,
				Headers:    map[string]string{"Content-Type": "text/plain"},
				Content:    "test content",
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "test content",
			expectedHeader: map[string]string{"Content-Type": "text/plain"},
		},
		{
			name: "file response",
			response: config.Response{
				File: testFilePath,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   testFileContent,
		},
		{
			name: "template response",
			response: config.Response{
				Content:  "Hello ${context.request.method}",
				Template: true,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Hello GET",
		},
		{
			name: "failure response",
			response: config.Response{
				Fail: "EmptyResponse",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name: "directory-based response with wildcard",
			response: config.Response{
				Dir: testDirPath,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"specific":"response"}`,
			expectedHeader: map[string]string{"Content-Type": "application/json"},
			requestPath:    "/api/responses/subdir/specific.json",
			requestMatcher: &config.RequestMatcher{
				Path: "/api/responses/*",
			},
		},
		{
			name: "directory-based response without wildcard",
			response: config.Response{
				Dir: testDirPath,
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Invalid directory",
			requestPath:    "/api/responses/specific.json",
			requestMatcher: &config.RequestMatcher{
				Path: "/api/responses/",
			},
		},
		{
			name: "directory-based response with nil request matcher",
			response: config.Response{
				Dir: testDirPath,
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Invalid directory",
			requestPath:    "/api/responses/specific.json",
		},
		{
			name: "directory-based response with empty request path",
			response: config.Response{
				Dir: testDirPath,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "index file content",
			expectedHeader: map[string]string{"Content-Type": "text/html; charset=utf-8"},
			requestPath:    "/api/responses/",
			requestMatcher: &config.RequestMatcher{
				Path: "/api/responses/*",
			},
		},
		{
			name: "directory-based response with non-existent file",
			response: config.Response{
				Dir: testDirPath,
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
			requestPath:    "/api/responses/nonexistent.json",
			requestMatcher: &config.RequestMatcher{
				Path: "/api/responses/*",
			},
		},
		{
			name: "directory-based response with non-existent directory",
			response: config.Response{
				Dir: "nonexistent",
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
			requestPath:    "/api/responses/specific.json",
			requestMatcher: &config.RequestMatcher{
				Path: "/api/responses/*",
			},
		},
		{
			name: "directory-based response with trailing slash uses index.html",
			response: config.Response{
				Dir: testDirPath,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "index file content",
			expectedHeader: map[string]string{"Content-Type": "text/html; charset=utf-8"},
			requestPath:    "/api/responses/",
			requestMatcher: &config.RequestMatcher{
				Path: "/api/responses/*",
			},
		},
		{
			name: "path traversal attempt is blocked",
			response: config.Response{
				File: "../../../etc/passwd",
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Invalid file path",
		},
		{
			name: "directory traversal attempt in dir response is blocked",
			response: config.Response{
				Dir: testDirPath,
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Invalid file path",
			requestPath:    "/api/responses/../../../etc/passwd",
			requestMatcher: &config.RequestMatcher{
				Path: "/api/responses/*",
			},
		},
		{
			name: "directory traversal attempt with encoded characters is blocked",
			response: config.Response{
				Dir: testDirPath,
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Invalid file path",
			requestPath:    "/api/responses/%2E%2E%2F%2E%2E%2F%2E%2E%2Fetc%2Fpasswd",
			requestMatcher: &config.RequestMatcher{
				Path: "/api/responses/*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := NewResponseState()
			reqPath := "/test"
			if tt.requestPath != "" {
				reqPath = tt.requestPath
			}
			req := httptest.NewRequest(http.MethodGet, reqPath, nil)
			requestStore := store.NewRequestStore()
			imposterConfig := &config.ImposterConfig{}
			exch := exchange.NewExchange(req, nil, requestStore, rs)
			processResponse(exch, tt.requestMatcher, &tt.response, tmpDir, imposterConfig)

			assert.Equal(t, tt.expectedStatus, rs.StatusCode)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, string(rs.Body))
			}
			for k, v := range tt.expectedHeader {
				assert.Equal(t, v, rs.Headers[k])
			}
		})
	}
}

func TestSetContentTypeHeader(t *testing.T) {
	tests := []struct {
		name                string
		fileNameHint        string
		expectedContentType string
	}{
		{
			name:                "markdown file",
			fileNameHint:        "readme.md",
			expectedContentType: "text/markdown",
		},
		{
			name:                "yaml file",
			fileNameHint:        "config.yaml",
			expectedContentType: "application/x-yaml",
		},
		{
			name:                "yml file",
			fileNameHint:        "config.yml",
			expectedContentType: "application/x-yaml",
		},
		{
			name:                "json file",
			fileNameHint:        "data.json",
			expectedContentType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &exchange.ResponseState{
				Headers: make(map[string]string),
				Body:    []byte("test content"),
			}
			SetContentTypeHeader(rs, tt.fileNameHint, "application/octet-stream", "application/json")
			assert.Equal(t, tt.expectedContentType, rs.Headers["Content-Type"])
		})
	}
}

// TestProcessResponseScriptPrecedence checks how values set explicitly on the
// response state, as a script does, interact with the resource's response
// configuration. Configured values act as defaults, so they only apply where
// the script has not already set something.
func TestProcessResponseScriptPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(tmpDir+"/config.txt", []byte("config file content"), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(tmpDir+"/script.txt", []byte("script file content"), 0644)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		setupState     func(rs *exchange.ResponseState)
		response       config.Response
		requestMatcher *config.RequestMatcher
		expectedStatus int
		expectedBody   string
		expectedHeader map[string]string
	}{
		{
			name:           "script status code overrides configured status code",
			setupState:     func(rs *exchange.ResponseState) { rs.SetStatusCode(http.StatusConflict) },
			response:       config.Response{StatusCode: http.StatusOK},
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "configured status code applies when the script sets none",
			setupState:     func(rs *exchange.ResponseState) {},
			response:       config.Response{StatusCode: http.StatusAccepted},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "configured status code does not apply to an unmarked default",
			setupState:     func(rs *exchange.ResponseState) { rs.StatusCode = http.StatusTeapot },
			response:       config.Response{StatusCode: http.StatusAccepted},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "script content overrides configured content",
			setupState:     func(rs *exchange.ResponseState) { rs.SetBody([]byte("script content")) },
			response:       config.Response{Content: "config content"},
			expectedStatus: http.StatusOK,
			expectedBody:   "script content",
		},
		{
			name:           "script content overrides configured file",
			setupState:     func(rs *exchange.ResponseState) { rs.SetBody([]byte("script content")) },
			response:       config.Response{File: "config.txt"},
			expectedStatus: http.StatusOK,
			expectedBody:   "script content",
		},
		{
			name: "script file overrides script content",
			setupState: func(rs *exchange.ResponseState) {
				rs.SetBody([]byte("script content"))
				rs.File = "script.txt"
			},
			response:       config.Response{Content: "config content"},
			expectedStatus: http.StatusOK,
			expectedBody:   "script file content",
		},
		{
			name:           "script file overrides configured file",
			setupState:     func(rs *exchange.ResponseState) { rs.File = "script.txt" },
			response:       config.Response{File: "config.txt"},
			expectedStatus: http.StatusOK,
			expectedBody:   "script file content",
		},
		{
			name:           "configured content applies when the script sets no body",
			setupState:     func(rs *exchange.ResponseState) { rs.SetStatusCode(http.StatusCreated) },
			response:       config.Response{Content: "config content"},
			expectedStatus: http.StatusCreated,
			expectedBody:   "config content",
		},
		{
			name:           "an empty body set by a script is not refilled from configuration",
			setupState:     func(rs *exchange.ResponseState) { rs.SetBody([]byte{}) },
			response:       config.Response{Content: "config content"},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "script content is templated when configuration enables templating",
			setupState:     func(rs *exchange.ResponseState) { rs.SetBody([]byte("method: ${context.request.method}")) },
			response:       config.Response{Template: true},
			expectedStatus: http.StatusOK,
			expectedBody:   "method: GET",
		},
		{
			name:           "script content is not templated when configuration does not enable templating",
			setupState:     func(rs *exchange.ResponseState) { rs.SetBody([]byte("method: ${context.request.method}")) },
			response:       config.Response{},
			expectedStatus: http.StatusOK,
			expectedBody:   "method: ${context.request.method}",
		},
		{
			name:           "script headers override configured headers of the same name",
			setupState:     func(rs *exchange.ResponseState) { rs.SetHeader("X-Shared", "script") },
			response:       config.Response{Headers: map[string]string{"X-Shared": "config"}},
			expectedStatus: http.StatusOK,
			expectedHeader: map[string]string{"X-Shared": "script"},
		},
		{
			name:           "script headers override configured headers differing only in casing",
			setupState:     func(rs *exchange.ResponseState) { rs.SetHeader("X-Shared", "script") },
			response:       config.Response{Headers: map[string]string{"x-shared": "config"}},
			expectedStatus: http.StatusOK,
			expectedHeader: map[string]string{"X-Shared": "script"},
		},
		{
			name:           "script headers are retained when configuration does not set them",
			setupState:     func(rs *exchange.ResponseState) { rs.SetHeader("X-Script", "script") },
			response:       config.Response{Headers: map[string]string{"X-Config": "config"}},
			expectedStatus: http.StatusOK,
			expectedHeader: map[string]string{"X-Script": "script", "X-Config": "config"},
		},
		{
			name:           "configured headers override those set other than by a script",
			setupState:     func(rs *exchange.ResponseState) { rs.Headers["Content-Type"] = "text/plain" },
			response:       config.Response{Headers: map[string]string{"Content-Type": "application/xml"}},
			expectedStatus: http.StatusOK,
			expectedHeader: map[string]string{"Content-Type": "application/xml"},
		},
		{
			name:           "script content overrides a directory response",
			setupState:     func(rs *exchange.ResponseState) { rs.SetBody([]byte("script content")) },
			response:       config.Response{Dir: "."},
			requestMatcher: &config.RequestMatcher{Path: "/no-wildcard"},
			expectedStatus: http.StatusOK,
			expectedBody:   "script content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := NewResponseState()
			tt.setupState(rs)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			exch := exchange.NewExchange(req, nil, store.NewRequestStore(), rs)
			processResponse(exch, tt.requestMatcher, &tt.response, tmpDir, &config.ImposterConfig{})

			assert.Equal(t, tt.expectedStatus, rs.StatusCode)
			assert.Equal(t, tt.expectedBody, string(rs.Body))
			for k, v := range tt.expectedHeader {
				assert.Equal(t, v, rs.Headers[k])
			}
		})
	}
}

// TestProcessResponseScriptDelayPrecedence checks that a delay set on the
// response state, as a script does, is used in preference to the delay in the
// response configuration.
func TestProcessResponseScriptDelayPrecedence(t *testing.T) {
	rs := NewResponseState()
	rs.Delay = config.Delay{Exact: 50}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	exch := exchange.NewExchange(req, nil, store.NewRequestStore(), rs)

	// the configured delay is an order of magnitude longer, so the elapsed
	// time shows which of the two was applied
	resp := config.Response{Delay: config.Delay{Exact: 1000}}

	start := time.Now()
	processResponse(exch, nil, &resp, t.TempDir(), &config.ImposterConfig{})
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond, "the delay set by the script should be applied")
	assert.Less(t, elapsed, 500*time.Millisecond, "the configured delay should not be applied")
}

// TestProcessResponseScriptFailurePrecedence checks that a failure set on the
// response state, as a script does, is simulated in preference to the failure
// in the response configuration, and short-circuits response processing.
func TestProcessResponseScriptFailurePrecedence(t *testing.T) {
	tests := []struct {
		name          string
		stateFailure  string
		configFailure string
		expectStopped bool
		expectBody    string
	}{
		{
			name:          "script failure takes precedence over configured failure",
			stateFailure:  "EmptyResponse",
			configFailure: "CloseConnection",
			expectStopped: false,
		},
		{
			name:          "configured failure applies when the script sets none",
			configFailure: "CloseConnection",
			expectStopped: true,
		},
		{
			name:         "script failure short-circuits configured content",
			stateFailure: "EmptyResponse",
			expectBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := NewResponseState()
			rs.Fail = tt.stateFailure

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			exch := exchange.NewExchange(req, nil, store.NewRequestStore(), rs)
			resp := config.Response{Fail: tt.configFailure, Content: "config content"}

			processResponse(exch, nil, &resp, t.TempDir(), &config.ImposterConfig{})

			assert.Equal(t, tt.expectStopped, rs.Stopped)
			assert.Equal(t, tt.expectBody, string(rs.Body))
		})
	}
}
