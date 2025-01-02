package rest

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
)

func TestHandler_HandleRequest_NoMatchingResource(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "imposter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := &config.Config{
		Plugin: "rest",
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Method: "GET",
					Path:   "/test",
				},
				Response: config.Response{
					Content: "test response",
				},
			},
		},
	}

	// Create handler
	handler, err := NewHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	// Handle request
	handler.HandleRequest(w, req)

	// Check response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandler_HandleRequest_MatchingResource(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "imposter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := &config.Config{
		Plugin: "rest",
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Method: "GET",
					Path:   "/test",
				},
				Response: config.Response{
					Content: "test response",
				},
			},
		},
	}

	// Create handler
	handler, err := NewHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Handle request
	handler.HandleRequest(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "test response" {
		t.Errorf("Expected response body %q, got %q", "test response", w.Body.String())
	}
}

func TestHandler_HandleRequest_WithInterceptor(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "imposter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := &config.Config{
		Plugin: "rest",
		Interceptors: []config.Interceptor{
			{
				RequestMatcher: config.RequestMatcher{
					Method: "GET",
					Path:   "/test",
				},
				Response: &config.Response{
					Content: "intercepted response",
				},
				Continue: false,
			},
		},
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Method: "GET",
					Path:   "/test",
				},
				Response: config.Response{
					Content: "test response",
				},
			},
		},
	}

	// Create handler
	handler, err := NewHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Handle request
	handler.HandleRequest(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "intercepted response" {
		t.Errorf("Expected response body %q, got %q", "intercepted response", w.Body.String())
	}
}

func TestHandler_HandleRequest_WithPathParams(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "imposter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := &config.Config{
		Plugin: "rest",
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Method: "GET",
					Path:   "/users/{id}",
					PathParams: map[string]config.MatcherUnmarshaler{
						"id": {
							Matcher: config.MatchCondition{
								Value:    "123",
								Operator: "EqualTo",
							},
						},
					},
				},
				Response: config.Response{
					Content: "user 123",
				},
			},
		},
	}

	// Create handler
	handler, err := NewHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	// Handle request
	handler.HandleRequest(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "user 123" {
		t.Errorf("Expected response body %q, got %q", "user 123", w.Body.String())
	}
}

func TestHandler_HandleRequest_WithResponseFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "imposter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test response file
	responseContent := "file response"
	responseFile := filepath.Join(tempDir, "response.txt")
	if err := os.WriteFile(responseFile, []byte(responseContent), 0644); err != nil {
		t.Fatalf("Failed to create response file: %v", err)
	}

	// Create test configuration
	cfg := &config.Config{
		Plugin: "rest",
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Method: "GET",
					Path:   "/test",
				},
				Response: config.Response{
					File: "response.txt",
				},
			},
		},
	}

	// Create handler
	handler, err := NewHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Handle request
	handler.HandleRequest(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != responseContent {
		t.Errorf("Expected response body %q, got %q", responseContent, w.Body.String())
	}
}

func TestHandler_HandleRequest_WithRequestBody(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "imposter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := &config.Config{
		Plugin: "rest",
		Resources: []config.Resource{
			{
				RequestMatcher: config.RequestMatcher{
					Method: "POST",
					Path:   "/test",
					RequestBody: config.RequestBody{
						BodyMatchCondition: config.BodyMatchCondition{
							MatchCondition: config.MatchCondition{
								Value:    "test body",
								Operator: "EqualTo",
							},
						},
					},
				},
				Response: config.Response{
					Content: "test response",
				},
			},
		},
	}

	// Create handler
	handler, err := NewHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString("test body"))
	w := httptest.NewRecorder()

	// Handle request
	handler.HandleRequest(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "test response" {
		t.Errorf("Expected response body %q, got %q", "test response", w.Body.String())
	}
}
