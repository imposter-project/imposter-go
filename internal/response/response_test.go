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
