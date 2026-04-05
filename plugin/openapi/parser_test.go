package openapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/imposter-project/imposter-go/pkg/feature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOpenAPIParser(t *testing.T) {
	tests := []struct {
		name          string
		specFile      string
		wantVersion   OpenAPIVersion
		wantOpCount   int
		wantErrSubstr string
	}{
		{
			name:        "OpenAPI 2.0 (Swagger)",
			specFile:    "testdata/v20/petstore20.yaml",
			wantVersion: OpenAPI20,
			wantOpCount: 20,
		},
		{
			name:        "OpenAPI 3.0",
			specFile:    "testdata/v30/petstore30.yaml",
			wantVersion: OpenAPI30,
			wantOpCount: 19,
		},
		// {
		// 	name:        "OpenAPI 3.1",
		// 	specFile:    "testdata/v31/petstore31.yaml",
		// 	wantVersion: OpenAPI31,
		// 	wantOpCount: 19,
		// },
		{
			name:          "Invalid spec file",
			specFile:      "testdata/nonexistent.yaml",
			wantErrSubstr: "cannot create new document",
		},
	}

	workingDir, _ := os.Getwd()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specFile := filepath.Join(workingDir, tt.specFile)
			opts := parserOptions{}
			parser, err := newOpenAPIParser(specFile, false, opts)

			if tt.wantErrSubstr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
				assert.Nil(t, parser)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, parser)
			assert.Equal(t, tt.wantVersion, parser.GetVersion())
			assert.Len(t, parser.GetOperations(), tt.wantOpCount)
		})
	}
}

func TestOpenAPIParser_ExternalURLRefsAreParsed(t *testing.T) {
	schemaJSON := `{
	  "User": {
		"properties": {
		  "id": {
			"type": "integer",
			"format": "int64",
			"example": 10
		  },
		  "username": {
			"type": "string",
			"example": "theUser"
		  },
		  "firstName": {
			"type": "string",
			"example": "John"
		  },
		  "lastName": {
			"type": "string",
			"example": "James"
		  }
		}
	  }
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/schemas/user.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(schemaJSON))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	workingDir, _ := os.Getwd()
	specFile := filepath.Join(workingDir, "testdata/externalRef/users.yaml")

	parser, err := newOpenAPIParser(specFile, false, parserOptions{
		externalReferenceBaseURL: ts.URL + "/",
	})

	assert.NoError(t, err)
	assert.NotNil(t, parser)
	assert.Equal(t, OpenAPI30, parser.GetVersion())
	assert.Len(t, parser.GetOperations(), 2)
}

// TestOpenAPIParser_FileRefsEnabledByDefault verifies that a spec whose
// schema is pulled in via a local-file $ref loads successfully under the
// default flag configuration (openapi.allowFileRefs = true).
func TestOpenAPIParser_FileRefsEnabledByDefault(t *testing.T) {
	feature.Reset()
	t.Cleanup(feature.Reset)

	workingDir, _ := os.Getwd()
	specFile := filepath.Join(workingDir, "testdata/externalFileRef/main.yaml")

	parser, err := newOpenAPIParser(specFile, false, parserOptions{})
	require.NoError(t, err)
	require.NotNil(t, parser)
	assert.Equal(t, OpenAPI30, parser.GetVersion())
	assert.Len(t, parser.GetOperations(), 1)
}

// TestOpenAPIParser_FileRefsDisabled verifies that file $ref resolution
// can be turned off via the feature flag, and that the resulting error
// is surfaced from newOpenAPIParser rather than leaking into a downstream
// panic.
func TestOpenAPIParser_FileRefsDisabled(t *testing.T) {
	feature.Reset()
	t.Setenv("IMPOSTER_OPENAPI_ALLOW_FILE_REFS", "false")
	t.Cleanup(feature.Reset)

	workingDir, _ := os.Getwd()
	specFile := filepath.Join(workingDir, "testdata/externalFileRef/main.yaml")

	// With file refs disabled, libopenapi fails to build a valid document
	// whose referenced schema can be resolved. We require either an error
	// from newOpenAPIParser or, at minimum, no panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("newOpenAPIParser must not panic when file refs are disabled, got: %v", r)
		}
	}()
	_, err := newOpenAPIParser(specFile, true, parserOptions{})
	assert.Error(t, err, "expected an error when file $ref resolution is disabled")
}

// TestOpenAPIParser_RemoteRefsOptIn reproduces imposter-project/imposter-go#36:
// a spec that references an absolute remote schema URL must fail cleanly
// by default (no panic) and resolve successfully once the operator opts in
// via IMPOSTER_OPENAPI_ALLOW_REMOTE_REFS=true.
func TestOpenAPIParser_RemoteRefsOptIn(t *testing.T) {
	schemaJSON := `{
	  "type": "object",
	  "properties": {
	    "id": {"type": "integer", "example": 10},
	    "username": {"type": "string", "example": "theUser"}
	  }
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/schemas/user.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(schemaJSON))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	spec := fmt.Sprintf(`openapi: 3.0.2
info:
  title: Remote ref test
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get all users
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '%s/schemas/user.json'
`, ts.URL)

	dir := t.TempDir()
	specFile := filepath.Join(dir, "remote-ref.yaml")
	if err := os.WriteFile(specFile, []byte(spec), 0o644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	t.Run("disabled by default fails cleanly", func(t *testing.T) {
		feature.Reset()
		t.Cleanup(feature.Reset)

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("newOpenAPIParser must not panic on remote ref when flag is off, got: %v", r)
			}
		}()
		_, err := newOpenAPIParser(specFile, true, parserOptions{})
		assert.Error(t, err, "expected an error when remote $ref resolution is disabled")
	})

	t.Run("opt-in flag allows resolution", func(t *testing.T) {
		feature.Reset()
		t.Setenv("IMPOSTER_OPENAPI_ALLOW_REMOTE_REFS", "true")
		t.Cleanup(feature.Reset)

		parser, err := newOpenAPIParser(specFile, false, parserOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, parser)
		assert.Len(t, parser.GetOperations(), 1)
	})
}
