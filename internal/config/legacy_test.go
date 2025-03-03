package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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
			name: "legacy config with staticData",
			config: `
plugin: rest
response:
  statusCode: 200
  staticData: example response`,
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
			name: "legacy config with resource level staticData",
			config: `
plugin: rest
resources:
  - path: /static-multi
    method: GET
    response:
      staticData: example response`,
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
			name: "legacy config with fail and delay",
			config: `
plugin: rest
response:
  fail: connection-reset
  delay:
    exact: 1000
    min: 500
    max: 2000`,
			expectedConfig: `
plugin: rest
resources:
- response:
    fail: connection-reset
    delay:
      exact: 1000
      min: 500
      max: 2000
    headers: {}`,
		},
		{
			name: "legacy config with dir and template",
			config: `
plugin: rest
response:
  dir: responses/dynamic
  template: true`,
			expectedConfig: `
plugin: rest
resources:
- response:
    dir: responses/dynamic
    template: true
    headers: {}`,
		},
		{
			name: "legacy config with SOAP and OpenAPI fields",
			config: `
plugin: rest
response:
  soapFault: true
  exampleName: error-response`,
			expectedConfig: `
plugin: rest
resources:
- response:
    soapFault: true
    exampleName: error-response
    headers: {}`,
		},
		{
			name: "legacy config with all response fields",
			config: `
plugin: rest
response:
  content: example response
  statusCode: 500
  dir: responses/errors
  file: error.json
  fail: connection-reset
  delay:
    exact: 1000
    min: 500
    max: 2000
  headers:
    Content-Type: application/json
  template: true
  soapFault: true
  exampleName: error-example`,
			expectedConfig: `
plugin: rest
resources:
- response:
    content: example response
    statusCode: 500
    dir: responses/errors
    file: error.json
    fail: connection-reset
    delay:
      exact: 1000
      min: 500
      max: 2000
    headers:
      Content-Type: application/json
    template: true
    soapFault: true
    exampleName: error-example`,
		},
		{
			name: "legacy config with resource level response fields",
			config: `
plugin: rest
resources:
  - path: /complex
    method: POST
    response:
      content: example response
      statusCode: 500
      dir: responses/errors
      file: error.json
      fail: connection-reset
      delay:
        exact: 1000
        min: 500
        max: 2000
      headers:
        Content-Type: application/json
      template: true
      soapFault: true
      exampleName: error-example`,
			expectedConfig: `
plugin: rest
resources:
- path: /complex
  method: POST
  response:
    content: example response
    statusCode: 500
    dir: responses/errors
    file: error.json
    fail: connection-reset
    delay:
      exact: 1000
      min: 500
      max: 2000
    headers:
      Content-Type: application/json
    template: true
    soapFault: true
    exampleName: error-example`,
		},
		{
			name:        "invalid yaml",
			config:      "invalid: [yaml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualConfig, err := transformLegacyConfig([]byte(tt.config))
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Normalize expected config for comparison
			var expectedConfig Config
			err = yaml.Unmarshal([]byte(tt.expectedConfig), &expectedConfig)
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
				assert.Equal(t, expectedConfig.Resources[i].Response.Dir, actualConfig.Resources[i].Response.Dir)
				assert.Equal(t, expectedConfig.Resources[i].Response.Fail, actualConfig.Resources[i].Response.Fail)
				assert.Equal(t, expectedConfig.Resources[i].Response.Delay, actualConfig.Resources[i].Response.Delay)
				assert.Equal(t, expectedConfig.Resources[i].Response.Headers, actualConfig.Resources[i].Response.Headers)
				assert.Equal(t, expectedConfig.Resources[i].Response.Template, actualConfig.Resources[i].Response.Template)
				assert.Equal(t, expectedConfig.Resources[i].Response.SoapFault, actualConfig.Resources[i].Response.SoapFault)
				assert.Equal(t, expectedConfig.Resources[i].Response.ExampleName, actualConfig.Resources[i].Response.ExampleName)

				// Compare steps
				assert.Equal(t, len(expectedConfig.Resources[i].Steps), len(actualConfig.Resources[i].Steps))
				if len(expectedConfig.Resources[i].Steps) > 0 {
					for j := range expectedConfig.Resources[i].Steps {
						assert.Equal(t, expectedConfig.Resources[i].Steps[j], actualConfig.Resources[i].Steps[j])
					}
				}

				// Compare security configuration
				if expectedConfig.Resources[i].Security != nil {
					if assert.NotNil(t, actualConfig.Resources[i].Security, "Security configuration should not be nil") {
						assert.Equal(t, expectedConfig.Resources[i].Security.Default, actualConfig.Resources[i].Security.Default)
						assert.Equal(t, len(expectedConfig.Resources[i].Security.Conditions), len(actualConfig.Resources[i].Security.Conditions))
						if len(expectedConfig.Resources[i].Security.Conditions) > 0 {
							for j := range expectedConfig.Resources[i].Security.Conditions {
								assert.Equal(t, expectedConfig.Resources[i].Security.Conditions[j].Effect, actualConfig.Resources[i].Security.Conditions[j].Effect)
								assert.Equal(t, expectedConfig.Resources[i].Security.Conditions[j].QueryParams, actualConfig.Resources[i].Security.Conditions[j].QueryParams)
								assert.Equal(t, expectedConfig.Resources[i].Security.Conditions[j].FormParams, actualConfig.Resources[i].Security.Conditions[j].FormParams)
								assert.Equal(t, expectedConfig.Resources[i].Security.Conditions[j].RequestHeaders, actualConfig.Resources[i].Security.Conditions[j].RequestHeaders)
							}
						}
					}
				} else {
					assert.Nil(t, actualConfig.Resources[i].Security, "Security configuration should be nil")
				}

				// Compare capture configuration
				if expectedConfig.Resources[i].Capture != nil {
					assert.Equal(t, expectedConfig.Resources[i].Capture, actualConfig.Resources[i].Capture)
				}
			}
		})
	}
}

func TestConvertColonPathToOpenAPI(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path with no parameters",
			path:     "/api/test",
			expected: "/api/test",
		},
		{
			name:     "path with one parameter",
			path:     "/api/:id",
			expected: "/api/{id}",
		},
		{
			name:     "path with multiple parameters",
			path:     "/:version/api/:resource/:id",
			expected: "/{version}/api/{resource}/{id}",
		},
		{
			name:     "path with mixed static and parameter segments",
			path:     "/api/v1/:resource/items/:id/details",
			expected: "/api/v1/{resource}/items/{id}/details",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "root path",
			path:     "/",
			expected: "/",
		},
		{
			name:     "path with trailing slash",
			path:     "/api/:resource/",
			expected: "/api/{resource}/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertColonPathToOpenAPI(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformLegacyConfigPathConversion(t *testing.T) {
	tests := []struct {
		name          string
		legacyConfig  string
		expectedPaths []string
		expectError   bool
	}{
		{
			name: "single resource with colon path",
			legacyConfig: `
plugin: rest
resources:
  - path: /api/:version/users/:id
    method: GET
    response:
      staticData: "test"`,
			expectedPaths: []string{"/api/{version}/users/{id}"},
			expectError:   false,
		},
		{
			name: "multiple resources with colon paths",
			legacyConfig: `
plugin: rest
resources:
  - path: /api/:version/users
    method: GET
  - path: /api/:version/users/:id
    method: GET
  - path: /static/path
    method: GET`,
			expectedPaths: []string{
				"/api/{version}/users",
				"/api/{version}/users/{id}",
				"/static/path",
			},
			expectError: false,
		},
		{
			name: "root level path with colon",
			legacyConfig: `
plugin: rest
path: /api/:version/test
method: GET
response:
  staticData: "test"`,
			expectedPaths: []string{"/api/{version}/test"},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := transformLegacyConfig([]byte(tt.legacyConfig))
			if tt.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, config)

			var paths []string
			for _, resource := range config.Resources {
				paths = append(paths, resource.Path)
			}
			assert.Equal(t, tt.expectedPaths, paths)
		})
	}
}
