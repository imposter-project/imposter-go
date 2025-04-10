package openapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSpecFile(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "imposter-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test spec file
	testSpecContent := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`)

	localSpecPath := filepath.Join(tmpDir, "test-spec.yaml")
	if err := os.WriteFile(localSpecPath, testSpecContent, 0644); err != nil {
		t.Fatalf("Failed to write test spec file: %v", err)
	}

	// Test cases
	tests := []struct {
		name      string
		specFile  string
		configDir string
		wantErr   bool
	}{
		{
			name:      "Local file with absolute path",
			specFile:  localSpecPath,
			configDir: "/some/dir",
			wantErr:   false,
		},
		{
			name:      "Local file with relative path",
			specFile:  "test-spec.yaml",
			configDir: tmpDir,
			wantErr:   false,
		},
		{
			name:      "Non-existent file",
			specFile:  "non-existent.yaml",
			configDir: tmpDir,
			wantErr:   false, // No error, file existence is checked later
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveSpecFile(tt.specFile, tt.configDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveSpecFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == "" {
					t.Errorf("resolveSpecFile() returned empty path")
				}
			}
		})
	}
}

func TestResolveSpecFileFromURL(t *testing.T) {
	// Create a test spec file
	testSpecContent := []byte(`openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}`)

	// Start a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/valid-spec.yaml" {
			w.WriteHeader(http.StatusOK)
			w.Write(testSpecContent)
		} else if r.URL.Path == "/not-found.yaml" {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Test cases
	tests := []struct {
		name     string
		specURL  string
		wantErr  bool
		checkTmp bool
	}{
		{
			name:     "Valid URL",
			specURL:  server.URL + "/valid-spec.yaml",
			wantErr:  false,
			checkTmp: true,
		},
		{
			name:     "URL with 404 response",
			specURL:  server.URL + "/not-found.yaml",
			wantErr:  true,
			checkTmp: false,
		},
		{
			name:     "URL with server error",
			specURL:  server.URL + "/server-error.yaml",
			wantErr:  true,
			checkTmp: false,
		},
		{
			name:     "Invalid URL",
			specURL:  "http://localhost:12345/non-existent-server",
			wantErr:  true,
			checkTmp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveSpecFile(tt.specURL, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveSpecFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkTmp {
				// Verify the file was downloaded correctly
				if result == "" {
					t.Errorf("resolveSpecFile() returned empty path")
					return
				}

				// Verify the file exists and has the correct content
				content, err := os.ReadFile(result)
				if err != nil {
					t.Errorf("Failed to read downloaded file: %v", err)
					return
				}

				if string(content) != string(testSpecContent) {
					t.Errorf("Downloaded file has incorrect content")
				}

				// Clean up temporary file
				os.Remove(result)
			}
		})
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "HTTP URL",
			input: "http://example.com/spec.yaml",
			want:  true,
		},
		{
			name:  "HTTPS URL",
			input: "https://example.com/spec.yaml",
			want:  true,
		},
		{
			name:  "Local absolute path",
			input: "/home/user/spec.yaml",
			want:  false,
		},
		{
			name:  "Local relative path",
			input: "./spec.yaml",
			want:  false,
		},
		{
			name:  "Empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isURL(tt.input); got != tt.want {
				t.Errorf("isURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
