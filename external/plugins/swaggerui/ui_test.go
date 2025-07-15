package main

import (
	"testing"
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
