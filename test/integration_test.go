package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
	"github.com/imposter-project/imposter-go/test/testutils"
)

func TestHomeRoute(t *testing.T) {
	configs := []config.Config{
		{
			Resources: []config.Resource{
				testutils.NewResource("GET", "/example", config.Response{
					StatusCode: 200,
					Content:    "Hello, world!",
				}),
			},
		},
	}
	imposterConfig := &config.ImposterConfig{
		ServerPort: "8080",
	}

	// Create a test request with an empty body
	req, err := http.NewRequest("GET", "/example", new(strings.Reader))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}

	rec := httptest.NewRecorder()
	handler.HandleRequest(rec, req, "", configs, imposterConfig)

	if status := rec.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	expectedBody := "Hello, world!"
	if rec.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, rec.Body.String())
	}
}

func TestIntegration_MatchJSONBody(t *testing.T) {
	configs := []config.Config{
		{
			Resources: []config.Resource{
				{
					RequestMatcher: config.RequestMatcher{
						Method: "POST",
						Path:   "/match-json",
						RequestBody: config.RequestBody{
							BodyMatchCondition: config.BodyMatchCondition{
								JSONPath: "$.user.name",
								MatchCondition: config.MatchCondition{
									Value: "Ada",
								},
							},
						},
					},
					Response: config.Response{
						StatusCode: 200,
						Content:    "Hello, Ada!",
					},
				},
			},
		},
	}
	imposterConfig := &config.ImposterConfig{
		ServerPort: "8080",
	}

	// Set up a test HTTP server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(w, r, "", configs, imposterConfig)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Create a test request
	resp, err := http.Post(server.URL+"/match-json", "application/json", strings.NewReader(`{"user": {"name": "Ada"}}`))
	if err != nil {
		t.Fatalf("Failed to make POST request: %v", err)
	}
	defer resp.Body.Close()

	// Validate response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}
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

	// Test with invalid user agent
	req, err := http.NewRequest("GET", "/example", new(strings.Reader))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("User-Agent", "Invalid-Agent")

	rec := httptest.NewRecorder()
	handler.HandleRequest(rec, req, "", configs, imposterConfig)

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
	handler.HandleRequest(rec, req, "", configs, imposterConfig)

	if status := rec.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	expectedBody = "Hello, world!"
	if rec.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, rec.Body.String())
	}
}

func TestInterceptors_Passthrough(t *testing.T) {
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
			Interceptors: []config.Interceptor{
				testutils.NewInterceptorWithCapture("GET", "/example", map[string]config.Capture{
					"userAgent": {
						Enabled: true,
						Store:   "request",
						CaptureKey: config.CaptureKey{
							RequestHeader: "User-Agent",
						},
					},
				}, true),
			},
		},
	}
	imposterConfig := &config.ImposterConfig{
		ServerPort: "8080",
	}

	req, err := http.NewRequest("GET", "/example", new(strings.Reader))
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}
	req.Header.Set("User-Agent", "Test-Agent")

	rec := httptest.NewRecorder()
	handler.HandleRequest(rec, req, "", configs, imposterConfig)

	if status := rec.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	expectedBody := "User agent: Test-Agent"
	if rec.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, rec.Body.String())
	}
}
