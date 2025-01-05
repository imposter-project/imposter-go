package response

import (
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
		responseState  *ResponseState
		expectedStatus int
		expectedBody   string
		expectedHeader map[string]string
	}{
		{
			name: "normal response",
			responseState: &ResponseState{
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
			responseState: &ResponseState{
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

	tests := []struct {
		name           string
		response       config.Response
		expectedStatus int
		expectedBody   string
		expectedHeader map[string]string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := NewResponseState()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			requestStore := make(store.Store)
			imposterConfig := &config.ImposterConfig{}

			ProcessResponse(rs, req, tt.response, tmpDir, requestStore, imposterConfig)

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
