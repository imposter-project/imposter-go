package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetServerURL(t *testing.T) {
	// Test with IMPOSTER_SERVER_URL set
	os.Setenv("IMPOSTER_SERVER_URL", "https://example.com:8080")
	defer os.Unsetenv("IMPOSTER_SERVER_URL")

	result := getServerURL()
	expected := "https://example.com:8080"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test with custom port
	os.Unsetenv("IMPOSTER_SERVER_URL")
	os.Setenv("IMPOSTER_PORT", "3000")
	defer os.Unsetenv("IMPOSTER_PORT")

	result = getServerURL()
	expected = "http://localhost:3000"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test with default port 80
	os.Setenv("IMPOSTER_PORT", "80")
	result = getServerURL()
	expected = "http://localhost"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestServeRawSpec_OpenAPI3(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create test OpenAPI 3.0 spec
	openapi3Spec := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/test": {
				"get": {
					"responses": {
						"200": {
							"description": "OK"
						}
					}
				}
			}
		}
	}`

	specFile := filepath.Join(tmpDir, "openapi.json")
	err := os.WriteFile(specFile, []byte(openapi3Spec), 0644)
	if err != nil {
		t.Fatalf("Failed to write test spec file: %v", err)
	}

	// Set up test environment
	os.Setenv("IMPOSTER_SERVER_URL", "https://test.example.com")
	defer os.Unsetenv("IMPOSTER_SERVER_URL")

	// Set up specConfigs
	specConfigs = []SpecConfig{
		{
			Name:         "openapi.json",
			URL:          "/_spec/openapi/openapi.json",
			OriginalPath: "openapi.json",
			ConfigDir:    tmpDir,
		},
	}

	// Test the function
	result := serveRawSpec("/_spec/openapi/openapi.json")

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	if result.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", result.Headers["Content-Type"])
	}

	// Parse the response body
	var responseSpec map[string]interface{}
	err = json.Unmarshal(result.Body, &responseSpec)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify that servers array was added
	servers, exists := responseSpec["servers"]
	if !exists {
		t.Error("Expected servers array to exist")
	}

	serverList, ok := servers.([]interface{})
	if !ok {
		t.Error("Expected servers to be an array")
	}

	if len(serverList) == 0 {
		t.Error("Expected at least one server entry")
	}

	firstServer, ok := serverList[0].(map[string]interface{})
	if !ok {
		t.Error("Expected first server to be an object")
	}

	serverURL, exists := firstServer["url"]
	if !exists {
		t.Error("Expected server URL to exist")
	}

	if serverURL != "https://test.example.com" {
		t.Errorf("Expected server URL to be 'https://test.example.com', got %s", serverURL)
	}
}

func TestServeRawSpec_Swagger2(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create test Swagger 2.0 spec
	swagger2Spec := `{
		"swagger": "2.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/test": {
				"get": {
					"responses": {
						"200": {
							"description": "OK"
						}
					}
				}
			}
		}
	}`

	specFile := filepath.Join(tmpDir, "swagger.json")
	err := os.WriteFile(specFile, []byte(swagger2Spec), 0644)
	if err != nil {
		t.Fatalf("Failed to write test spec file: %v", err)
	}

	// Set up test environment
	os.Setenv("IMPOSTER_SERVER_URL", "https://test.example.com/api/v1")
	defer os.Unsetenv("IMPOSTER_SERVER_URL")

	// Set up specConfigs
	specConfigs = []SpecConfig{
		{
			Name:         "swagger.json",
			URL:          "/_spec/openapi/swagger.json",
			OriginalPath: "swagger.json",
			ConfigDir:    tmpDir,
		},
	}

	// Test the function
	result := serveRawSpec("/_spec/openapi/swagger.json")

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	// Parse the response body
	var responseSpec map[string]interface{}
	err = json.Unmarshal(result.Body, &responseSpec)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify that host and basePath were set
	host, exists := responseSpec["host"]
	if !exists {
		t.Error("Expected host to exist")
	}

	if host != "test.example.com" {
		t.Errorf("Expected host to be 'test.example.com', got %s", host)
	}

	basePath, exists := responseSpec["basePath"]
	if !exists {
		t.Error("Expected basePath to exist")
	}

	if basePath != "/api/v1" {
		t.Errorf("Expected basePath to be '/api/v1', got %s", basePath)
	}

	// Verify that schemes was set to https
	schemes, exists := responseSpec["schemes"]
	if !exists {
		t.Error("Expected schemes to exist")
	}

	schemeList, ok := schemes.([]interface{})
	if !ok {
		t.Error("Expected schemes to be an array")
	}

	if len(schemeList) == 0 || schemeList[0] != "https" {
		t.Error("Expected first scheme to be 'https'")
	}
}

func TestServeRawSpec_YAML(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create test YAML OpenAPI 3.0 spec
	yamlSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: OK`

	specFile := filepath.Join(tmpDir, "openapi.yaml")
	err := os.WriteFile(specFile, []byte(yamlSpec), 0644)
	if err != nil {
		t.Fatalf("Failed to write test spec file: %v", err)
	}

	// Set up test environment
	os.Setenv("IMPOSTER_SERVER_URL", "http://localhost:8080")
	defer os.Unsetenv("IMPOSTER_SERVER_URL")

	// Set up specConfigs
	specConfigs = []SpecConfig{
		{
			Name:         "openapi.yaml",
			URL:          "/_spec/openapi/openapi.yaml",
			OriginalPath: "openapi.yaml",
			ConfigDir:    tmpDir,
		},
	}

	// Test the function
	result := serveRawSpec("/_spec/openapi/openapi.yaml")

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	// Parse the response body
	var responseSpec map[string]interface{}
	err = json.Unmarshal(result.Body, &responseSpec)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify that servers array was added
	servers, exists := responseSpec["servers"]
	if !exists {
		t.Error("Expected servers array to exist")
	}

	serverList, ok := servers.([]interface{})
	if !ok {
		t.Error("Expected servers to be an array")
	}

	if len(serverList) == 0 {
		t.Error("Expected at least one server entry")
	}

	firstServer, ok := serverList[0].(map[string]interface{})
	if !ok {
		t.Error("Expected first server to be an object")
	}

	serverURL, exists := firstServer["url"]
	if !exists {
		t.Error("Expected server URL to exist")
	}

	if serverURL != "http://localhost:8080" {
		t.Errorf("Expected server URL to be 'http://localhost:8080', got %s", serverURL)
	}
}

func TestServeRawSpec_NotFound(t *testing.T) {
	// Set up empty specConfigs
	specConfigs = []SpecConfig{}

	// Test with non-existent spec
	result := serveRawSpec("/_spec/openapi/nonexistent.json")

	if result != nil {
		t.Error("Expected nil result for non-existent spec")
	}
}
