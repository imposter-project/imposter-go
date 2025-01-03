package rest

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/imposter-project/imposter-go/internal/response"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
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

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if responseState.Handled {
		t.Error("Expected response to not be handled for no match")
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

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled for match")
	}

	if responseState.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseState.StatusCode)
	}

	if string(responseState.Body) != "test response" {
		t.Errorf("Expected response body %q, got %q", "test response", string(responseState.Body))
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

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled for interceptor")
	}

	if responseState.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseState.StatusCode)
	}

	if string(responseState.Body) != "intercepted response" {
		t.Errorf("Expected response body %q, got %q", "intercepted response", string(responseState.Body))
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

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled for path params match")
	}

	if responseState.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseState.StatusCode)
	}

	if string(responseState.Body) != "user 123" {
		t.Errorf("Expected response body %q, got %q", "user 123", string(responseState.Body))
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

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled for file response")
	}

	if responseState.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseState.StatusCode)
	}

	if string(responseState.Body) != responseContent {
		t.Errorf("Expected response body %q, got %q", responseContent, string(responseState.Body))
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

	// Initialise store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle request
	handler.HandleRequest(req, requestStore, responseState)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled for request body")
	}

	if responseState.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseState.StatusCode)
	}

	if string(responseState.Body) != "test response" {
		t.Errorf("Expected response body %q, got %q", "test response", string(responseState.Body))
	}
}
