package main

import (
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/external/shared"
)

func TestServeStaticContent_EmptyPath(t *testing.T) {
	// Test the specific case where path is empty after prefix removal
	originalSpecPrefix := specPrefixPath
	defer func() { specPrefixPath = originalSpecPrefix }()

	// Set up test environment
	specPrefixPath = "/_spec"

	// Test empty path after prefix removal
	result := serveStaticContent("/_spec")

	// Verify redirect response
	if result.StatusCode != 302 {
		t.Errorf("Expected status code 302, got %d", result.StatusCode)
	}

	expectedLocation := specPrefixPath + "/"
	if result.Headers["Location"] != expectedLocation {
		t.Errorf("Expected Location header '%s', got '%s'", expectedLocation, result.Headers["Location"])
	}
}

func TestServeStaticContent_EmptyPathWithCustomPrefix(t *testing.T) {
	// Test with custom prefix
	originalSpecPrefix := specPrefixPath
	defer func() { specPrefixPath = originalSpecPrefix }()

	specPrefixPath = "/custom"

	result := serveStaticContent("/custom")

	if result.StatusCode != 302 {
		t.Errorf("Expected status code 302, got %d", result.StatusCode)
	}

	expectedLocation := "/custom/"
	if result.Headers["Location"] != expectedLocation {
		t.Errorf("Expected Location header '%s', got '%s'", expectedLocation, result.Headers["Location"])
	}
}

func TestServeStaticContent_EmptyPathWithServerBasePath(t *testing.T) {
	// When IMPOSTER_SERVER_URL includes a base path (e.g. behind a reverse
	// proxy), the trailing-slash redirect must include that base path so the
	// browser resolves it against the proxy rather than the origin root.
	originalSpecPrefix := specPrefixPath
	originalConfig := config
	defer func() { specPrefixPath = originalSpecPrefix; config = originalConfig }()

	specPrefixPath = "/_spec"
	config = shared.ExternalConfig{Server: shared.ServerConfig{URL: "http://localhost:8080/myapp"}}

	result := serveStaticContent("/_spec")

	if result.StatusCode != 302 {
		t.Errorf("Expected status code 302, got %d", result.StatusCode)
	}

	expectedLocation := "/myapp/_spec/"
	if result.Headers["Location"] != expectedLocation {
		t.Errorf("Expected Location header '%s', got '%s'", expectedLocation, result.Headers["Location"])
	}
}

func TestGenerateInitialiser_SpecUrlServerBasePath(t *testing.T) {
	originalSpecConfigs := specConfigs
	originalConfig := config
	originalResp := initialiserResp
	defer func() {
		specConfigs = originalSpecConfigs
		config = originalConfig
		initialiserResp = originalResp
	}()

	specConfigs = []SpecConfig{{Name: "petstore.yaml", URL: "/_spec/openapi/petstore.yaml"}}

	tests := []struct {
		name        string
		serverURL   string
		expectedURL string
	}{
		{
			name:        "no base path",
			serverURL:   "http://localhost:8080",
			expectedURL: `"url":"/_spec/openapi/petstore.yaml"`,
		},
		{
			name:        "with base path",
			serverURL:   "http://localhost:8080/myapp",
			expectedURL: `"url":"/myapp/_spec/openapi/petstore.yaml"`,
		},
		{
			name:        "base path with trailing slash",
			serverURL:   "http://localhost:8080/myapp/",
			expectedURL: `"url":"/myapp/_spec/openapi/petstore.yaml"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config = shared.ExternalConfig{Server: shared.ServerConfig{URL: tt.serverURL}}

			if err := generateInitialiser(); err != nil {
				t.Fatalf("generateInitialiser returned error: %v", err)
			}

			body := string(initialiserResp.Body)
			if !strings.Contains(body, tt.expectedURL) {
				t.Errorf("expected initialiser to contain %s, got body: %s", tt.expectedURL, body)
			}
		})
	}
}
