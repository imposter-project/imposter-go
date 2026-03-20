package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/imposter-project/imposter-go/external/shared"
)

func TestGenerateWSDLConfig(t *testing.T) {
	// Reset global state
	wsdlConfigs = nil

	configs := []shared.LightweightConfig{
		{
			ConfigDir: "/tmp/test",
			Plugin:    "soap",
			WSDLFile:  "petstore.wsdl",
		},
		{
			ConfigDir: "/tmp/test2",
			Plugin:    "openapi",
			SpecFile:  "openapi.yaml",
		},
		{
			ConfigDir: "/tmp/test3",
			Plugin:    "soap",
			WSDLFile:  "service.wsdl",
		},
	}

	err := generateWSDLConfig(configs)
	if err != nil {
		t.Fatalf("generateWSDLConfig failed: %v", err)
	}

	if len(wsdlConfigs) != 2 {
		t.Fatalf("Expected 2 WSDL configs, got %d", len(wsdlConfigs))
	}

	if wsdlConfigs[0].Label != "petstore.wsdl" {
		t.Errorf("Expected label 'petstore.wsdl', got '%s'", wsdlConfigs[0].Label)
	}
	if wsdlConfigs[0].URL != "/_wsdl/wsdl/petstore.wsdl" {
		t.Errorf("Expected URL '/_wsdl/wsdl/petstore.wsdl', got '%s'", wsdlConfigs[0].URL)
	}

	if wsdlConfigs[1].Label != "service.wsdl" {
		t.Errorf("Expected label 'service.wsdl', got '%s'", wsdlConfigs[1].Label)
	}
}

func TestGenerateWSDLConfig_SkipsEmptyWSDLFile(t *testing.T) {
	wsdlConfigs = nil

	configs := []shared.LightweightConfig{
		{
			ConfigDir: "/tmp/test",
			Plugin:    "soap",
			WSDLFile:  "",
		},
	}

	err := generateWSDLConfig(configs)
	if err != nil {
		t.Fatalf("generateWSDLConfig failed: %v", err)
	}

	if len(wsdlConfigs) != 0 {
		t.Errorf("Expected 0 WSDL configs, got %d", len(wsdlConfigs))
	}
}

func TestServeRawWSDL(t *testing.T) {
	tmpDir := t.TempDir()

	wsdlContent := `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             name="PetStoreService"
             targetNamespace="urn:com:example:petstore">
  <message name="getPetByIdRequest">
    <part name="id" type="xsd:int"/>
  </message>
  <message name="getPetByIdResponse">
    <part name="pet" element="tns:Pet"/>
  </message>
</definitions>`

	wsdlFile := filepath.Join(tmpDir, "petstore.wsdl")
	err := os.WriteFile(wsdlFile, []byte(wsdlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test WSDL file: %v", err)
	}

	// Reset global state
	wsdlConfigs = []WSDLConfig{
		{
			Label:        "petstore.wsdl",
			URL:          "/_wsdl/wsdl/petstore.wsdl",
			OriginalPath: "petstore.wsdl",
			ConfigDir:    tmpDir,
		},
	}
	cachedWSDLs = make(map[string][]byte)

	result := serveRawWSDL("/_wsdl/wsdl/petstore.wsdl")

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", result.StatusCode)
	}

	if result.Headers["Content-Type"] != "application/xml" {
		t.Errorf("Expected Content-Type application/xml, got %s", result.Headers["Content-Type"])
	}

	if string(result.Body) != wsdlContent {
		t.Errorf("Response body doesn't match WSDL content")
	}
}

func TestServeRawWSDL_NotFound(t *testing.T) {
	wsdlConfigs = []WSDLConfig{}

	result := serveRawWSDL("/_wsdl/wsdl/nonexistent.wsdl")

	if result != nil {
		t.Error("Expected nil result for non-existent WSDL")
	}
}

func TestServeRawWSDL_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	wsdlConfigs = []WSDLConfig{
		{
			Label:        "missing.wsdl",
			URL:          "/_wsdl/wsdl/missing.wsdl",
			OriginalPath: "missing.wsdl",
			ConfigDir:    tmpDir,
		},
	}
	cachedWSDLs = make(map[string][]byte)

	result := serveRawWSDL("/_wsdl/wsdl/missing.wsdl")

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", result.StatusCode)
	}
}

func TestServeRawWSDL_Cached(t *testing.T) {
	tmpDir := t.TempDir()

	wsdlContent := `<?xml version="1.0" encoding="UTF-8"?><definitions/>`

	wsdlFile := filepath.Join(tmpDir, "cached.wsdl")
	err := os.WriteFile(wsdlFile, []byte(wsdlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test WSDL file: %v", err)
	}

	wsdlConfigs = []WSDLConfig{
		{
			Label:        "cached.wsdl",
			URL:          "/_wsdl/wsdl/cached.wsdl",
			OriginalPath: "cached.wsdl",
			ConfigDir:    tmpDir,
		},
	}
	cachedWSDLs = make(map[string][]byte)

	// First call populates cache
	result1 := serveRawWSDL("/_wsdl/wsdl/cached.wsdl")
	if result1 == nil || result1.StatusCode != 200 {
		t.Fatal("Expected 200 on first call")
	}

	// Delete the file to prove the cache is used
	os.Remove(wsdlFile)

	// Second call should use cache
	result2 := serveRawWSDL("/_wsdl/wsdl/cached.wsdl")
	if result2 == nil || result2.StatusCode != 200 {
		t.Fatal("Expected 200 on second (cached) call")
	}

	if string(result2.Body) != wsdlContent {
		t.Error("Cached response body doesn't match")
	}
}
