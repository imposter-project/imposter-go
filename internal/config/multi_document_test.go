package config

import (
	"os"
	"testing"
)

func TestParseMultipleYAMLDocuments(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		expectedDocs int
		expectError  bool
		validate     func(*testing.T, []Config)
	}{
		{
			name: "single document",
			yamlContent: `plugin: rest
resources:
  - path: /test
    method: GET
    response:
      statusCode: 200`,
			expectedDocs: 1,
			expectError:  false,
			validate: func(t *testing.T, configs []Config) {
				if configs[0].Plugin != "rest" {
					t.Errorf("Expected plugin 'rest', got '%s'", configs[0].Plugin)
				}
				if len(configs[0].Resources) != 1 {
					t.Errorf("Expected 1 resource, got %d", len(configs[0].Resources))
				}
			},
		},
		{
			name: "multiple documents with OpenAPI and SwaggerUI",
			yamlContent: `plugin: openapi
specFile: api.json
resources:
  - path: /pets
    method: GET
    response:
      statusCode: 200
      body: '{"pets": []}'
---
plugin: swaggerui
config:
  specUrl: http://localhost:8080/system/openapi
  theme: "dark"
  title: "Pet Store API Documentation"`,
			expectedDocs: 2,
			expectError:  false,
			validate: func(t *testing.T, configs []Config) {
				// First document - OpenAPI
				if configs[0].Plugin != "openapi" {
					t.Errorf("Expected first plugin 'openapi', got '%s'", configs[0].Plugin)
				}
				if configs[0].SpecFile != "api.json" {
					t.Errorf("Expected specFile 'api.json', got '%s'", configs[0].SpecFile)
				}
				if len(configs[0].Resources) != 1 {
					t.Errorf("Expected 1 resource in first config, got %d", len(configs[0].Resources))
				}

				// Second document - SwaggerUI
				if configs[1].Plugin != "swaggerui" {
					t.Errorf("Expected second plugin 'swaggerui', got '%s'", configs[1].Plugin)
				}

				// Check that plugin config is captured
				if configs[1].PluginConfig.Kind == 0 {
					t.Error("Expected second config to have plugin config")
				}

				// Verify plugin config content can be unmarshaled
				var pluginConfig map[string]interface{}
				if err := configs[1].PluginConfig.Decode(&pluginConfig); err != nil {
					t.Errorf("Failed to unmarshal plugin config: %v", err)
				}

				if specUrl, ok := pluginConfig["specUrl"].(string); !ok || specUrl != "http://localhost:8080/system/openapi" {
					t.Errorf("Expected specUrl 'http://localhost:8080/system/openapi', got '%v'", pluginConfig["specUrl"])
				}
			},
		},
		{
			name: "three documents with mixed plugins",
			yamlContent: `plugin: rest
resources:
  - path: /health
    method: GET
    response:
      statusCode: 200
      body: '{"status": "ok"}'
---
plugin: openapi
specFile: petstore.yaml
---
plugin: swaggerui
config:
  title: "Multi-Service API"`,
			expectedDocs: 3,
			expectError:  false,
			validate: func(t *testing.T, configs []Config) {
				expectedPlugins := []string{"rest", "openapi", "swaggerui"}
				for i, expectedPlugin := range expectedPlugins {
					if configs[i].Plugin != expectedPlugin {
						t.Errorf("Expected plugin %d to be '%s', got '%s'", i, expectedPlugin, configs[i].Plugin)
					}
				}
			},
		},
		{
			name:         "empty document",
			yamlContent:  ``,
			expectedDocs: 0,
			expectError:  true,
		},
		{
			name: "invalid YAML",
			yamlContent: `plugin: rest
resources:
  - path: /test
    method: GET
    response:
      statusCode: 200
---
plugin: openapi
specFile: [invalid yaml structure`,
			expectedDocs: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs, err := parseMultipleYAMLDocuments([]byte(tt.yamlContent))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(configs) != tt.expectedDocs {
				t.Errorf("Expected %d documents, got %d", tt.expectedDocs, len(configs))
				return
			}

			if tt.validate != nil {
				tt.validate(t, configs)
			}
		})
	}
}

func TestParseConfigMultiDocument(t *testing.T) {
	// Test that parseConfig properly handles multi-document files
	// by creating a temporary file and parsing it

	multiDocYAML := `plugin: rest
resources:
  - path: /api/v1/status
    method: GET
    response:
      statusCode: 200
      body: '{"status": "running"}'
---
plugin: openapi
specFile: status-api.yaml
basePath: /api/v1`

	// Create temporary test directory and file
	tmpDir := t.TempDir()
	configFile := tmpDir + "/test-config.yaml"

	if err := writeFile(configFile, []byte(multiDocYAML)); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	imposterConfig := &ImposterConfig{
		LegacyConfigSupported: false,
		ServerPort:            "8080",
		ServerUrl:             "http://localhost:8080",
	}

	configs, err := parseConfig(configFile, imposterConfig)
	if err != nil {
		t.Fatalf("Failed to parse multi-document config: %v", err)
	}

	if len(configs) != 2 {
		t.Errorf("Expected 2 configurations, got %d", len(configs))
	}

	// Verify first config
	if configs[0].Plugin != "rest" {
		t.Errorf("Expected first plugin 'rest', got '%s'", configs[0].Plugin)
	}
	if len(configs[0].Resources) != 1 {
		t.Errorf("Expected 1 resource in first config, got %d", len(configs[0].Resources))
	}

	// Verify second config
	if configs[1].Plugin != "openapi" {
		t.Errorf("Expected second plugin 'openapi', got '%s'", configs[1].Plugin)
	}
	if configs[1].SpecFile != "status-api.yaml" {
		t.Errorf("Expected specFile 'status-api.yaml', got '%s'", configs[1].SpecFile)
	}
	if configs[1].BasePath != "/api/v1" {
		t.Errorf("Expected basePath '/api/v1', got '%s'", configs[1].BasePath)
	}
}

// Helper function to write files (similar to os.WriteFile but ensures it works in test environment)
func writeFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}
