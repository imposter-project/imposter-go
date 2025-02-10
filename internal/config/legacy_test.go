package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIsLegacyConfig(t *testing.T) {
	// Set up environment for testing
	os.Setenv("IMPOSTER_SUPPORT_LEGACY_CONFIG", "true")
	defer os.Unsetenv("IMPOSTER_SUPPORT_LEGACY_CONFIG")

	tests := []struct {
		name     string
		config   string
		expected bool
		envVar   string
	}{
		{
			name: "legacy config with file",
			config: `
plugin: rest
response:
  statusCode: 200
  file: example.json`,
			expected: true,
			envVar:   "true",
		},
		{
			name: "legacy config with content",
			config: `
plugin: rest
response:
  statusCode: 200
  content: example response`,
			expected: true,
			envVar:   "true",
		},
		{
			name: "legacy config with staticContent",
			config: `
plugin: rest
response:
  statusCode: 200
  staticContent: example response`,
			expected: true,
			envVar:   "true",
		},
		{
			name: "legacy config with staticFile",
			config: `
plugin: rest
response:
  statusCode: 200
  staticFile: example.json`,
			expected: true,
			envVar:   "true",
		},
		{
			name: "legacy config with scriptFile",
			config: `
plugin: rest
response:
  statusCode: 200
  scriptFile: example.js`,
			expected: true,
			envVar:   "true",
		},
		{
			name: "legacy config with root level properties",
			config: `
plugin: rest
path: /static-multi
contentType: text/html
method: GET`,
			expected: true,
			envVar:   "true",
		},
		{
			name: "legacy config with resource level contentType",
			config: `
plugin: rest
resources:
  - path: /static-multi
    contentType: text/html
    method: GET`,
			expected: true,
			envVar:   "true",
		},
		{
			name: "current format config",
			config: `
plugin: rest
resources:
- response:
    statusCode: 200
    file: example.json`,
			expected: false,
			envVar:   "true",
		},
		{
			name: "legacy support disabled",
			config: `
plugin: rest
response:
  statusCode: 200
  file: example.json`,
			expected: false,
			envVar:   "false",
		},
		{
			name:     "invalid yaml",
			config:   "invalid: [yaml",
			expected: false,
			envVar:   "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("IMPOSTER_SUPPORT_LEGACY_CONFIG", tt.envVar)
			result := isLegacyConfig([]byte(tt.config))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformLegacyConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         string
		expectedConfig string
		expectError    bool
	}{
		{
			name: "legacy config with file",
			config: `
plugin: rest
response:
  statusCode: 200
  file: example.json`,
			expectedConfig: `
plugin: rest
resources:
- response:
    statusCode: 200
    file: example.json
    headers: {}`,
		},
		{
			name: "legacy config with content",
			config: `
plugin: rest
response:
  statusCode: 200
  content: example response`,
			expectedConfig: `
plugin: rest
resources:
- response:
    statusCode: 200
    content: example response
    headers: {}`,
		},
		{
			name: "legacy config with staticContent",
			config: `
plugin: rest
response:
  statusCode: 200
  staticContent: example response`,
			expectedConfig: `
plugin: rest
resources:
- response:
    statusCode: 200
    content: example response
    headers: {}`,
		},
		{
			name: "legacy config with staticFile",
			config: `
plugin: rest
response:
  statusCode: 200
  staticFile: example.json`,
			expectedConfig: `
plugin: rest
resources:
- response:
    statusCode: 200
    file: example.json
    headers: {}`,
		},
		{
			name: "legacy config with headers",
			config: `
plugin: rest
response:
  statusCode: 200
  content: example response
  headers:
    Content-Type: application/json`,
			expectedConfig: `
plugin: rest
resources:
- response:
    statusCode: 200
    content: example response
    headers:
      Content-Type: application/json`,
		},
		{
			name: "legacy config with root level properties",
			config: `
plugin: rest
path: /static-multi
contentType: text/html
method: GET`,
			expectedConfig: `
plugin: rest
resources:
- path: /static-multi
  method: GET
  response:
    headers:
      Content-Type: text/html`,
		},
		{
			name: "legacy config with resource level contentType",
			config: `
plugin: rest
resources:
  - path: /static-multi
    contentType: text/html
    method: GET`,
			expectedConfig: `
plugin: rest
resources:
- path: /static-multi
  method: GET
  response:
    headers:
      Content-Type: text/html`,
		},
		{
			name: "legacy config with resource level staticContent",
			config: `
plugin: rest
resources:
  - path: /static-multi
    method: GET
    response:
      staticContent: example response`,
			expectedConfig: `
plugin: rest
resources:
- path: /static-multi
  method: GET
  response:
    content: example response
    headers: {}`,
		},
		{
			name: "legacy config with root level scriptFile",
			config: `
plugin: rest
path: /script
response:
  scriptFile: script.js`,
			expectedConfig: `
plugin: rest
resources:
- path: /script
  steps:
  - type: script
    lang: javascript
    file: script.js
  response:
    headers: {}`,
		},
		{
			name: "legacy config with resource level scriptFile",
			config: `
plugin: rest
resources:
  - path: /script
    method: POST
    response:
      scriptFile: script.js`,
			expectedConfig: `
plugin: rest
resources:
- path: /script
  method: POST
  steps:
  - type: script
    lang: javascript
    file: script.js
  response:
    headers: {}`,
		},
		{
			name: "legacy config with scriptFile and other response fields",
			config: `
plugin: rest
resources:
  - path: /script
    response:
      scriptFile: script.js
      headers:
        Content-Type: application/json
      statusCode: 201`,
			expectedConfig: `
plugin: rest
resources:
- path: /script
  steps:
  - type: script
    lang: javascript
    file: script.js
  response:
    statusCode: 201
    headers:
      Content-Type: application/json`,
		},
		{
			name: "legacy config with both scriptFile and staticFile",
			config: `
plugin: rest
resources:
  - path: /script-and-file
    response:
      scriptFile: script.js
      staticFile: data.json
      headers:
        Content-Type: application/json
      statusCode: 201`,
			expectedConfig: `
plugin: rest
resources:
- path: /script-and-file
  steps:
  - type: script
    lang: javascript
    file: script.js
  response:
    statusCode: 201
    file: data.json
    headers:
      Content-Type: application/json`,
		},
		{
			name:        "invalid yaml",
			config:      "invalid: [yaml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformLegacyConfig([]byte(tt.config))
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Normalize expected and actual configs for comparison
			var expectedConfig, actualConfig Config
			err = yaml.Unmarshal([]byte(tt.expectedConfig), &expectedConfig)
			require.NoError(t, err)
			err = yaml.Unmarshal(result, &actualConfig)
			require.NoError(t, err)

			assert.Equal(t, expectedConfig.Plugin, actualConfig.Plugin)
			assert.Equal(t, len(expectedConfig.Resources), len(actualConfig.Resources))

			// Compare resources
			for i := range expectedConfig.Resources {
				assert.Equal(t, expectedConfig.Resources[i].Path, actualConfig.Resources[i].Path)
				assert.Equal(t, expectedConfig.Resources[i].Method, actualConfig.Resources[i].Method)
				assert.Equal(t, expectedConfig.Resources[i].Response.StatusCode, actualConfig.Resources[i].Response.StatusCode)
				assert.Equal(t, expectedConfig.Resources[i].Response.Content, actualConfig.Resources[i].Response.Content)
				assert.Equal(t, expectedConfig.Resources[i].Response.File, actualConfig.Resources[i].Response.File)
				assert.Equal(t, expectedConfig.Resources[i].Response.Headers, actualConfig.Resources[i].Response.Headers)
				assert.Equal(t, len(expectedConfig.Resources[i].Steps), len(actualConfig.Resources[i].Steps))
				if len(expectedConfig.Resources[i].Steps) > 0 {
					for j := range expectedConfig.Resources[i].Steps {
						assert.Equal(t, expectedConfig.Resources[i].Steps[j].Type, actualConfig.Resources[i].Steps[j].Type)
						assert.Equal(t, expectedConfig.Resources[i].Steps[j].Lang, actualConfig.Resources[i].Steps[j].Lang)
						assert.Equal(t, expectedConfig.Resources[i].Steps[j].File, actualConfig.Resources[i].Steps[j].File)
					}
				}
			}
		})
	}
}
