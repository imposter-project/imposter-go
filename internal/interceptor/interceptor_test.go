package interceptor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
)

// mockProcessor implements response.Processor for testing
type mockProcessor struct {
	processedResponses []config.Response
}

func (m *mockProcessor) ProcessResponse(rs *response.ResponseState, r *http.Request, resp config.Response, requestStore store.Store) {
	m.processedResponses = append(m.processedResponses, resp)
	rs.StatusCode = resp.StatusCode
	rs.Body = []byte(resp.Content)
}

func TestProcessInterceptor(t *testing.T) {
	tests := []struct {
		name               string
		interceptor        config.Interceptor
		wantContinue       bool
		wantResponseCount  int
		wantCaptureCount   int
		wantResponseStatus int
	}{
		{
			name: "interceptor with response and continue",
			interceptor: config.Interceptor{
				Response: &config.Response{
					StatusCode: 200,
					Content:    "test response",
				},
				Continue: true,
			},
			wantContinue:       true,
			wantResponseCount:  1,
			wantCaptureCount:   0,
			wantResponseStatus: 200,
		},
		{
			name: "interceptor with response and no continue",
			interceptor: config.Interceptor{
				Response: &config.Response{
					StatusCode: 403,
					Content:    "forbidden",
				},
				Continue: false,
			},
			wantContinue:       false,
			wantResponseCount:  1,
			wantCaptureCount:   0,
			wantResponseStatus: 403,
		},
		{
			name: "interceptor with capture only",
			interceptor: config.Interceptor{
				RequestMatcher: config.RequestMatcher{
					Capture: map[string]config.Capture{
						"test": {
							Enabled: true,
							Store:   "request",
							CaptureKey: config.CaptureKey{
								RequestHeader: "X-Test",
							},
						},
					},
				},
				Continue: true,
			},
			wantContinue:      true,
			wantResponseCount: 0,
			wantCaptureCount:  1,
		},
		{
			name: "interceptor with both capture and response",
			interceptor: config.Interceptor{
				RequestMatcher: config.RequestMatcher{
					Capture: map[string]config.Capture{
						"test": {
							Enabled: true,
							Store:   "request",
							CaptureKey: config.CaptureKey{
								RequestHeader: "X-Test",
							},
						},
					},
				},
				Response: &config.Response{
					StatusCode: 200,
					Content:    "test with capture",
				},
				Continue: true,
			},
			wantContinue:       true,
			wantResponseCount:  1,
			wantCaptureCount:   1,
			wantResponseStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			processor := &mockProcessor{}
			requestStore := make(store.Store)
			responseState := response.NewResponseState()

			// Create test request
			req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test body"))
			req.Header.Set("X-Test", "test-value")

			// Run test
			gotContinue := ProcessInterceptor(
				responseState,
				req,
				[]byte("test body"),
				tt.interceptor,
				requestStore,
				&config.ImposterConfig{},
				".",
				processor,
			)

			// Verify results
			if gotContinue != tt.wantContinue {
				t.Errorf("ProcessInterceptor() continue = %v, want %v", gotContinue, tt.wantContinue)
			}

			if len(processor.processedResponses) != tt.wantResponseCount {
				t.Errorf("ProcessInterceptor() processed %d responses, want %d", len(processor.processedResponses), tt.wantResponseCount)
			}

			if tt.interceptor.Response != nil {
				if responseState.StatusCode != tt.wantResponseStatus {
					t.Errorf("ProcessInterceptor() status = %d, want %d", responseState.StatusCode, tt.wantResponseStatus)
				}
			}

			if len(tt.interceptor.Capture) > 0 {
				value, exists := requestStore["test"]
				if !exists {
					t.Error("ProcessInterceptor() did not store capture data")
				}
				if exists && value != "test-value" {
					t.Errorf("ProcessInterceptor() captured wrong value: got %v, want test-value", value)
				}
			}
		})
	}
}
