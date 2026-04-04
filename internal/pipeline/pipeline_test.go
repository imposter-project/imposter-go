package pipeline

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
)

// newTestExchange builds an Exchange + response state for a GET request at path.
func newTestExchange(path string) (*exchange.Exchange, *exchange.ResponseState) {
	req := httptest.NewRequest("GET", path, nil)
	requestStore := store.NewRequestStore()
	responseState := response.NewResponseState()
	exch := exchange.NewExchange(req, nil, requestStore, responseState)
	return exch, responseState
}

// TestRunPipeline_InterceptorContinueTrue verifies that an interceptor with
// Continue: true runs its response processing and then the pipeline proceeds
// to resource matching (rather than short-circuiting). This branch is not
// covered by the rest/soap handler tests, which only exercise Continue: false.
func TestRunPipeline_InterceptorContinueTrue(t *testing.T) {
	cfg := &config.Config{
		Plugin: "rest",
		Interceptors: []config.Interceptor{
			{
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "GET",
						Path:   "/test",
					},
					Response: &config.Response{Content: "from interceptor"},
				},
				Continue: true,
			},
		},
		Resources: []config.Resource{
			{
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "GET",
						Path:   "/test",
					},
					Response: &config.Response{Content: "from resource"},
				},
			},
		},
	}

	exch, responseState := newTestExchange("/test")

	// Track ProcessResponse hook invocations to verify ordering and
	// prove the hook dispatch path is wired correctly.
	var calls []string
	hooks := &ProtocolHooks{
		ProcessResponse: func(
			e *exchange.Exchange,
			rm *config.RequestMatcher,
			resp *config.Response,
			rp response.Processor,
		) {
			calls = append(calls, resp.Content)
			// Mirror the default behaviour so the final body is observable.
			rp(e, rm, resp)
		},
	}

	respProc := response.NewProcessor(&config.ImposterConfig{}, "")
	RunPipeline(cfg, &config.ImposterConfig{}, exch, respProc, hooks)

	if len(calls) != 2 {
		t.Fatalf("expected ProcessResponse to be called twice (interceptor + resource), got %d: %v", len(calls), calls)
	}
	if calls[0] != "from interceptor" {
		t.Errorf("expected first call to be interceptor response, got %q", calls[0])
	}
	if calls[1] != "from resource" {
		t.Errorf("expected second call to be resource response, got %q", calls[1])
	}
	if !responseState.Handled {
		t.Error("expected response to be marked handled after resource match")
	}
	if string(responseState.Body) != "from resource" {
		t.Errorf("expected final body to be resource response, got %q", string(responseState.Body))
	}
}

// TestRunPipeline_StepError_DefaultHandler verifies that a failing step on a
// matched resource triggers the default step error handler, which writes a
// 500 response and marks the exchange as handled. Not covered by rest/soap
// tests.
func TestRunPipeline_StepError_DefaultHandler(t *testing.T) {
	cfg := &config.Config{
		Plugin: "rest",
		Resources: []config.Resource{
			{
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "GET",
						Path:   "/test",
					},
					Steps: []config.Step{
						{Type: "bogus-step-type"}, // RunSteps returns an error for unknown type
					},
					Response: &config.Response{Content: "should not reach"},
				},
			},
		},
	}

	exch, responseState := newTestExchange("/test")
	respProc := response.NewProcessor(&config.ImposterConfig{}, "")

	RunPipeline(cfg, &config.ImposterConfig{}, exch, respProc, nil)

	if !responseState.Handled {
		t.Error("expected response to be handled after step error")
	}
	if responseState.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, responseState.StatusCode)
	}
	if !strings.Contains(string(responseState.Body), "Failed to execute steps") {
		t.Errorf("expected body to mention step failure, got %q", string(responseState.Body))
	}
	if string(responseState.Body) == "should not reach" {
		t.Error("resource response must not be processed when steps fail")
	}
}

// TestRunPipeline_StepError_CustomHook verifies that when a protocol provides
// a custom OnStepError hook, it is invoked instead of the default handler.
// This is the extension point the gRPC plugin relies on to emit a gRPC-status
// error instead of an HTTP 500.
func TestRunPipeline_StepError_CustomHook(t *testing.T) {
	cfg := &config.Config{
		Plugin: "rest",
		Resources: []config.Resource{
			{
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "GET",
						Path:   "/test",
					},
					Steps: []config.Step{
						{Type: "bogus-step-type"},
					},
				},
			},
		},
	}

	exch, responseState := newTestExchange("/test")
	respProc := response.NewProcessor(&config.ImposterConfig{}, "")

	var hookCalled bool
	var hookMsg string
	hooks := &ProtocolHooks{
		OnStepError: func(rs *exchange.ResponseState, msg string) {
			hookCalled = true
			hookMsg = msg
			rs.StatusCode = http.StatusTeapot // sentinel so we can distinguish from default 500
			rs.Body = []byte("custom error body")
			rs.Handled = true
		},
	}

	RunPipeline(cfg, &config.ImposterConfig{}, exch, respProc, hooks)

	if !hookCalled {
		t.Fatal("expected custom OnStepError hook to be invoked")
	}
	if hookMsg != "Failed to execute steps" {
		t.Errorf("expected hook message %q, got %q", "Failed to execute steps", hookMsg)
	}
	if responseState.StatusCode != http.StatusTeapot {
		t.Errorf("expected custom status %d, got %d (custom hook did not override default)",
			http.StatusTeapot, responseState.StatusCode)
	}
	if string(responseState.Body) != "custom error body" {
		t.Errorf("expected custom body, got %q", string(responseState.Body))
	}
}
