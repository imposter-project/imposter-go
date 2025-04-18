package rest

import (
	"bytes"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "GET",
						Path:   "/test",
					},
					Response: &config.Response{
						Content: "test response",
					},
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/nonexistent", nil)

	// Initialise store and response state
	requestStore := store.NewRequestStore()
	responseState := response.NewResponseState()
	exch := exchange.NewExchangeFromRequest(req, nil, requestStore)

	// Handle request
	handler.HandleRequest(exch, nil)

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
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "GET",
						Path:   "/test",
					},
					Response: &config.Response{
						Content: "test response",
					},
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)

	// Initialise store and response state
	requestStore := store.NewRequestStore()
	responseState := response.NewResponseState()
	responseProc := response.NewProcessor(&config.ImposterConfig{}, tempDir)
	exch := exchange.NewExchange(req, nil, requestStore, responseState)

	// Handle request
	handler.HandleRequest(exch, responseProc)

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
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "GET",
						Path:   "/test",
					},
					Response: &config.Response{
						Content: "intercepted response",
					},
				},
				Continue: false,
			},
		},
		Resources: []config.Resource{
			{
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "GET",
						Path:   "/test",
					},
					Response: &config.Response{
						Content: "test response",
					},
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)

	// Initialise store and response state
	requestStore := store.NewRequestStore()
	responseState := response.NewResponseState()
	responseProc := response.NewProcessor(&config.ImposterConfig{}, tempDir)
	exch := exchange.NewExchange(req, nil, requestStore, responseState)

	// Handle request
	handler.HandleRequest(exch, responseProc)

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
				BaseResource: config.BaseResource{
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
					Response: &config.Response{
						Content: "user 123",
					},
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/users/123", nil)

	// Initialise store and response state
	requestStore := store.NewRequestStore()
	responseState := response.NewResponseState()
	responseProc := response.NewProcessor(&config.ImposterConfig{}, tempDir)
	exch := exchange.NewExchange(req, nil, requestStore, responseState)

	// Handle request
	handler.HandleRequest(exch, responseProc)

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
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "GET",
						Path:   "/test",
					},
					Response: &config.Response{
						File: "response.txt",
					},
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)

	// Initialise store and response state
	requestStore := store.NewRequestStore()
	responseState := response.NewResponseState()
	responseProc := response.NewProcessor(&config.ImposterConfig{}, tempDir)
	exch := exchange.NewExchange(req, nil, requestStore, responseState)

	// Handle request
	handler.HandleRequest(exch, responseProc)

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
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "POST",
						Path:   "/test",
						RequestBody: config.RequestBody{
							BodyMatchCondition: &config.BodyMatchCondition{
								MatchCondition: config.MatchCondition{
									Value:    "test body",
									Operator: "EqualTo",
								},
							},
						},
					},
					Response: &config.Response{
						Content: "test response",
					},
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	body := "test body"
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))

	// Initialise store and response state
	requestStore := store.NewRequestStore()
	responseState := response.NewResponseState()
	responseProc := response.NewProcessor(&config.ImposterConfig{}, tempDir)
	exch := exchange.NewExchange(req, []byte(body), requestStore, responseState)

	// Handle request
	handler.HandleRequest(exch, responseProc)

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

func TestHandler_HandleRequest_WithXMLNamespaces(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "imposter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := &config.Config{
		Plugin: "rest",
		System: &config.System{
			XMLNamespaces: map[string]string{
				"ns": "http://example.com",
			},
		},
		Resources: []config.Resource{
			{
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{
						Method: "POST",
						Path:   "/test",
						RequestBody: config.RequestBody{
							BodyMatchCondition: &config.BodyMatchCondition{
								MatchCondition: config.MatchCondition{
									Value: "Grace",
								},
								XPath: "//ns:user",
							},
						},
					},
					Response: &config.Response{
						Content: "test response",
					},
				},
			},
		},
	}

	// Create handler
	handler, err := NewPluginHandler(cfg, tempDir, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create test request
	xmlBody := `<?xml version="1.0" encoding="UTF-8"?>
<root xmlns:ns="http://example.com">
    <ns:user>Grace</ns:user>
</root>`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(xmlBody))
	req.Header.Set("Content-Type", "application/xml")

	// Initialise store and response state
	requestStore := store.NewRequestStore()
	responseState := response.NewResponseState()
	responseProc := response.NewProcessor(&config.ImposterConfig{}, tempDir)
	exch := exchange.NewExchange(req, []byte(xmlBody), requestStore, responseState)

	// Handle request
	handler.HandleRequest(exch, responseProc)

	// Check response
	if !responseState.Handled {
		t.Error("Expected response to be handled for XML namespace match")
	}

	if responseState.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, responseState.StatusCode)
	}

	if string(responseState.Body) != "test response" {
		t.Errorf("Expected response body %q, got %q", "test response", string(responseState.Body))
	}
}
