package main

import (
	"testing"
)

func TestServeStaticContent_EmptyPath(t *testing.T) {
	originalPrefix := wsdlPrefixPath
	defer func() { wsdlPrefixPath = originalPrefix }()

	wsdlPrefixPath = "/_wsdl"

	result := serveStaticContent("/_wsdl")

	if result.StatusCode != 302 {
		t.Errorf("Expected status code 302, got %d", result.StatusCode)
	}

	expectedLocation := "/_wsdl/"
	if result.Headers["Location"] != expectedLocation {
		t.Errorf("Expected Location header '%s', got '%s'", expectedLocation, result.Headers["Location"])
	}
}

func TestServeStaticContent_EmptyPathWithCustomPrefix(t *testing.T) {
	originalPrefix := wsdlPrefixPath
	defer func() { wsdlPrefixPath = originalPrefix }()

	wsdlPrefixPath = "/custom"

	result := serveStaticContent("/custom")

	if result.StatusCode != 302 {
		t.Errorf("Expected status code 302, got %d", result.StatusCode)
	}

	expectedLocation := "/custom/"
	if result.Headers["Location"] != expectedLocation {
		t.Errorf("Expected Location header '%s', got '%s'", expectedLocation, result.Headers["Location"])
	}
}

func TestServeStaticContent_IndexHtml(t *testing.T) {
	originalPrefix := wsdlPrefixPath
	defer func() { wsdlPrefixPath = originalPrefix }()

	wsdlPrefixPath = "/_wsdl"

	result := serveStaticContent("/_wsdl/")

	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}
}

func TestServeStaticContent_NotFound(t *testing.T) {
	originalPrefix := wsdlPrefixPath
	defer func() { wsdlPrefixPath = originalPrefix }()

	wsdlPrefixPath = "/_wsdl"

	result := serveStaticContent("/_wsdl/nonexistent.xyz")

	if result.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", result.StatusCode)
	}
}

func TestServeStaticContent_Initialiser(t *testing.T) {
	originalPrefix := wsdlPrefixPath
	defer func() { wsdlPrefixPath = originalPrefix }()

	wsdlPrefixPath = "/_wsdl"

	// Set up the initialiser response
	wsdlConfigs = nil
	err := generateInitialiser()
	if err != nil {
		t.Fatalf("Failed to generate initialiser: %v", err)
	}

	result := serveStaticContent("/_wsdl/wsdl-initializer.js")

	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	body := string(result.Body)
	if len(body) == 0 {
		t.Error("Expected non-empty initialiser body")
	}
}

func TestGenerateInitialiser(t *testing.T) {
	wsdlConfigs = []WSDLConfig{
		{Label: "petstore.wsdl", URL: "/_wsdl/wsdl/petstore.wsdl"},
		{Label: "service.wsdl", URL: "/_wsdl/wsdl/service.wsdl"},
	}
	config.Server.URL = "http://localhost:8080"

	err := generateInitialiser()
	if err != nil {
		t.Fatalf("generateInitialiser failed: %v", err)
	}

	if initialiserResp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", initialiserResp.StatusCode)
	}

	body := string(initialiserResp.Body)
	if body == "" {
		t.Error("Expected non-empty body")
	}

	// Should contain the WSDL config data
	if !contains(body, "petstore.wsdl") {
		t.Error("Expected body to contain 'petstore.wsdl'")
	}
	if !contains(body, "service.wsdl") {
		t.Error("Expected body to contain 'service.wsdl'")
	}
	if !contains(body, "WsdlWeb.init(") {
		t.Error("Expected body to contain 'WsdlWeb.init('")
	}
	if !contains(body, "baseUrlOverride: 'http://localhost:8080'") {
		t.Error("Expected body to contain baseUrlOverride with server URL")
	}
}

func TestGenerateInitialiser_EmptyConfigs(t *testing.T) {
	wsdlConfigs = nil

	err := generateInitialiser()
	if err != nil {
		t.Fatalf("generateInitialiser failed: %v", err)
	}

	body := string(initialiserResp.Body)
	if !contains(body, "WsdlWeb.init(") {
		t.Error("Expected body to contain 'WsdlWeb.init('")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
