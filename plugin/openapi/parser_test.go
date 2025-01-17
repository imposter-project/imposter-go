package openapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
			parser, err := newOpenAPIParser(specFile)

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
