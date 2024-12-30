package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
)

func TestHomeRoute(t *testing.T) {
	configs := []config.Config{
		{
			Resources: []config.Resource{
				{
					Method: "GET",
					Path:   "/example",
					Response: config.Response{
						StatusCode: 200,
						Content:    "Hello, world!",
					},
				},
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
