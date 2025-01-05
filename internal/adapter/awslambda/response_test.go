package awslambda

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponseRecorder_Header(t *testing.T) {
	recorder := &responseRecorder{
		Headers: make(http.Header),
	}

	// Test getting headers
	headers := recorder.Header()
	assert.NotNil(t, headers)

	// Test modifying headers
	headers.Set("Content-Type", "application/json")
	assert.Equal(t, "application/json", recorder.Headers.Get("Content-Type"))
}

func TestResponseRecorder_Write(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus bool
		data          []byte
		expectedLen   int
		expectedBody  string
		expectedCode  int
	}{
		{
			name:          "write with no status set",
			initialStatus: false,
			data:          []byte("test data"),
			expectedLen:   9,
			expectedBody:  "test data",
			expectedCode:  http.StatusOK,
		},
		{
			name:          "write with status already set",
			initialStatus: true,
			data:          []byte("more data"),
			expectedLen:   9,
			expectedBody:  "more data",
			expectedCode:  http.StatusCreated,
		},
		{
			name:          "write empty data",
			initialStatus: true,
			data:          []byte{},
			expectedLen:   0,
			expectedBody:  "",
			expectedCode:  http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &responseRecorder{
				Headers: make(http.Header),
			}

			if tt.initialStatus {
				recorder.WriteHeader(http.StatusCreated)
			}

			n, err := recorder.Write(tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedLen, n)
			assert.Equal(t, tt.expectedBody, recorder.Body.String())
			assert.Equal(t, tt.expectedCode, recorder.StatusCode)
			assert.True(t, recorder.writtenStatus)
		})
	}
}

func TestResponseRecorder_WriteHeader(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  int
		secondStatus   int
		expectedStatus int
	}{
		{
			name:           "write OK status then Not Found",
			initialStatus:  http.StatusOK,
			secondStatus:   http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "write Created status then OK",
			initialStatus:  http.StatusCreated,
			secondStatus:   http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "write Not Found status then Created",
			initialStatus:  http.StatusNotFound,
			secondStatus:   http.StatusCreated,
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &responseRecorder{
				Headers: make(http.Header),
			}

			// Write initial status
			recorder.WriteHeader(tt.initialStatus)
			assert.Equal(t, tt.initialStatus, recorder.StatusCode)
			assert.True(t, recorder.writtenStatus)

			// Writing a different status should change the status code
			recorder.WriteHeader(tt.secondStatus)
			assert.Equal(t, tt.expectedStatus, recorder.StatusCode)
			assert.True(t, recorder.writtenStatus)
		})
	}
}

func TestResponseRecorder_Integration(t *testing.T) {
	recorder := &responseRecorder{
		Headers: make(http.Header),
	}

	// Test the complete flow of using the recorder
	recorder.Header().Set("Content-Type", "text/plain")
	recorder.Header().Set("X-Custom-Header", "test-value")

	// Write some data without setting status first
	n1, err := recorder.Write([]byte("first "))
	assert.NoError(t, err)
	assert.Equal(t, 6, n1)
	assert.Equal(t, http.StatusOK, recorder.StatusCode)

	// Write more data
	n2, err := recorder.Write([]byte("second"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n2)

	// Verify final state
	assert.Equal(t, "first second", recorder.Body.String())
	assert.Equal(t, "text/plain", recorder.Headers.Get("Content-Type"))
	assert.Equal(t, "test-value", recorder.Headers.Get("X-Custom-Header"))
	assert.True(t, recorder.writtenStatus)
}
