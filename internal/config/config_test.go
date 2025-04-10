package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig_SOAP(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test WSDL file
	wsdlContent := `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/">
    <!-- Test WSDL content -->
</wsdl:definitions>`
	err := os.WriteFile(filepath.Join(tempDir, "test.wsdl"), []byte(wsdlContent), 0644)
	require.NoError(t, err)

	// Create a test config file
	configContent := `plugin: soap
wsdlFile: test.wsdl
resources:
  - path: /test
    operation: testOperation
    soapAction: testAction
    response:
      content: test response
      statusCode: 200
  - path: /another
    operation: anotherOperation
    response:
      content: another response
      statusCode: 200`

	err = os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	configs := LoadConfig(tempDir, &ImposterConfig{})
	require.Len(t, configs, 1)

	cfg := configs[0]
	require.Equal(t, "soap", cfg.Plugin)
	require.Equal(t, "test.wsdl", cfg.WSDLFile)
	require.Len(t, cfg.Resources, 2)

	// Check first resource
	require.Equal(t, "/test", cfg.Resources[0].Path)
	require.Equal(t, "testOperation", cfg.Resources[0].Operation)
	require.Equal(t, "testAction", cfg.Resources[0].SOAPAction)
	require.Equal(t, "test response", cfg.Resources[0].Response.Content)
	require.Equal(t, 200, cfg.Resources[0].Response.StatusCode)

	// Check second resource
	require.Equal(t, "/another", cfg.Resources[1].Path)
	require.Equal(t, "anotherOperation", cfg.Resources[1].Operation)
	require.Equal(t, "another response", cfg.Resources[1].Response.Content)
	require.Equal(t, 200, cfg.Resources[1].Response.StatusCode)
}

func TestLoadConfig_WithCapture(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	imposterConfig := &ImposterConfig{}

	// Create a test config file with capture configuration
	configContent := `plugin: rest
resources:
  - path: /test
    capture:
      user_id:
        enabled: true
        store: users
        pathParam: id
      request_data:
        enabled: true
        store: requests
        requestBody:
          jsonPath: $.data
          xmlNamespaces:
            ns: http://example.com
      header_value:
        enabled: true
        store: headers
        requestHeader: X-Custom-Header
      query_value:
        enabled: true
        store: queries
        queryParam: filter
      form_value:
        enabled: true
        store: forms
        formParam: field
      const_value:
        enabled: true
        store: constants
        const: fixed_value
      expr_value:
        enabled: true
        store: expressions
        expression: request.method + "_" + request.path
    response:
      content: test response
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	configs := LoadConfig(tempDir, imposterConfig)
	require.Len(t, configs, 1)

	cfg := configs[0]
	require.Equal(t, "rest", cfg.Plugin)
	require.Len(t, cfg.Resources, 1)

	// Check capture configurations
	resource := cfg.Resources[0]
	require.Len(t, resource.Capture, 7)

	// Check path parameter capture
	userIDCapture := resource.Capture["user_id"]
	require.NotNil(t, userIDCapture.Enabled)
	require.True(t, *userIDCapture.Enabled)
	require.Equal(t, "users", userIDCapture.Store)
	require.Equal(t, "id", userIDCapture.PathParam)

	// Check request body capture
	requestDataCapture := resource.Capture["request_data"]
	require.NotNil(t, requestDataCapture.Enabled)
	require.True(t, *requestDataCapture.Enabled)
	require.Equal(t, "requests", requestDataCapture.Store)
	require.Equal(t, "$.data", requestDataCapture.RequestBody.JSONPath)
	require.Equal(t, "http://example.com", requestDataCapture.RequestBody.XMLNamespaces["ns"])

	// Check header capture
	headerCapture := resource.Capture["header_value"]
	require.NotNil(t, headerCapture.Enabled)
	require.True(t, *headerCapture.Enabled)
	require.Equal(t, "headers", headerCapture.Store)
	require.Equal(t, "X-Custom-Header", headerCapture.RequestHeader)

	// Check query parameter capture
	queryCapture := resource.Capture["query_value"]
	require.NotNil(t, queryCapture.Enabled)
	require.True(t, *queryCapture.Enabled)
	require.Equal(t, "queries", queryCapture.Store)
	require.Equal(t, "filter", queryCapture.QueryParam)

	// Check form parameter capture
	formCapture := resource.Capture["form_value"]
	require.NotNil(t, formCapture.Enabled)
	require.True(t, *formCapture.Enabled)
	require.Equal(t, "forms", formCapture.Store)
	require.Equal(t, "field", formCapture.FormParam)

	// Check constant capture
	constCapture := resource.Capture["const_value"]
	require.NotNil(t, constCapture.Enabled)
	require.True(t, *constCapture.Enabled)
	require.Equal(t, "constants", constCapture.Store)
	require.Equal(t, "fixed_value", constCapture.Const)

	// Check expression capture
	exprCapture := resource.Capture["expr_value"]
	require.NotNil(t, exprCapture.Enabled)
	require.True(t, *exprCapture.Enabled)
	require.Equal(t, "expressions", exprCapture.Store)
	require.Equal(t, "request.method + \"_\" + request.path", exprCapture.Expression)
}

func TestLoadConfig_WithEnvVars(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_HOST", "example.com")
	os.Setenv("TEST_PORT", "8080")
	os.Setenv("TEST_PATH", "/api/v1")
	defer func() {
		os.Unsetenv("TEST_HOST")
		os.Unsetenv("TEST_PORT")
		os.Unsetenv("TEST_PATH")
	}()

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	imposterConfig := &ImposterConfig{}

	// Create a test config file with environment variables
	configContent := `plugin: rest
basePath: ${env.TEST_PATH}
resources:
  - path: /test
    response:
      content: Response from ${env.TEST_HOST}:${env.TEST_PORT}
      statusCode: 200
      headers:
        Host: ${env.TEST_HOST}
        X-Test: ${env.TEST_PATH}`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	configs := LoadConfig(tempDir, imposterConfig)
	require.Len(t, configs, 1)

	cfg := configs[0]
	require.Equal(t, "/api/v1", cfg.BasePath)
	require.Len(t, cfg.Resources, 1)

	resource := cfg.Resources[0]
	require.Equal(t, "Response from example.com:8080", resource.Response.Content)
	require.Equal(t, "example.com", resource.Response.Headers["Host"])
	require.Equal(t, "/api/v1", resource.Response.Headers["X-Test"])
}

func TestLoadConfig_WithAutoBasePath(t *testing.T) {
	// Set up auto base path and recursive scanning environment variables
	os.Setenv("IMPOSTER_AUTO_BASE_PATH", "true")
	os.Setenv("IMPOSTER_CONFIG_SCAN_RECURSIVE", "true")
	defer func() {
		os.Unsetenv("IMPOSTER_AUTO_BASE_PATH")
		os.Unsetenv("IMPOSTER_CONFIG_SCAN_RECURSIVE")
	}()

	// Create a temporary directory structure for test files
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "api", "v1")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	imposterConfig := &ImposterConfig{}

	// Create test config files in different directories
	rootConfig := `plugin: rest
resources:
  - path: /test
    response:
      content: root response
      statusCode: 200`

	subConfig := `plugin: rest
resources:
  - path: /users
    response:
      file: users-response.json
      statusCode: 200
  
  - path: /static
    response:
      dir: static-content

system:
  stores:
    users:
      preloadFile: user-db.json
`

	err = os.WriteFile(filepath.Join(tempDir, "root-config.yaml"), []byte(rootConfig), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "api-config.yaml"), []byte(subConfig), 0644)
	require.NoError(t, err)

	// Load the configs
	configs := LoadConfig(tempDir, imposterConfig)
	require.Len(t, configs, 2)

	// Check root config
	var rootCfg, subCfg *Config
	for i := range configs {
		if configs[i].BasePath == "" || configs[i].BasePath == "." {
			rootCfg = &configs[i]
		} else if configs[i].BasePath == "/api/v1" {
			subCfg = &configs[i]
		}
	}

	require.NotNil(t, rootCfg, "Root config not found")
	require.NotNil(t, subCfg, "Sub-directory config not found")

	// Check root config paths
	require.Len(t, rootCfg.Resources, 1)
	require.Equal(t, "/test", rootCfg.Resources[0].Path)

	// Check sub-directory config paths
	require.Len(t, subCfg.Resources, 2)
	require.Equal(t, "/api/v1/users", subCfg.Resources[0].Path)
	require.Equal(t, "api/v1/users-response.json", subCfg.Resources[0].Response.File)
	require.Equal(t, "api/v1/static-content", subCfg.Resources[1].Response.Dir)

	// Check system store preload file path
	require.Equal(t, "api/v1/user-db.json", subCfg.System.Stores["users"].PreloadFile)
}

func TestLoadConfig_WithInterceptors(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	imposterConfig := &ImposterConfig{}

	// Create a test config file with interceptors
	configContent := `plugin: rest
interceptors:
  - path: /auth
    method: POST
    requestHeaders:
      Authorization:
        value: Bearer
        operator: Contains
    response:
      content: Unauthorized
      statusCode: 401
      headers:
        WWW-Authenticate: Bearer realm="test"
    continue: false
  - path: /metrics
    method: GET
    response:
      content: Metrics collected
      statusCode: 200
    continue: true
resources:
  - path: /test
    response:
      content: test response
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	configs := LoadConfig(tempDir, imposterConfig)
	require.Len(t, configs, 1)

	cfg := configs[0]
	require.Equal(t, "rest", cfg.Plugin)
	require.Len(t, cfg.Interceptors, 2)
	require.Len(t, cfg.Resources, 1)

	// Check first interceptor
	auth := cfg.Interceptors[0]
	require.Equal(t, "/auth", auth.Path)
	require.Equal(t, "POST", auth.Method)
	require.Contains(t, auth.RequestHeaders, "Authorization")
	authMatcher, ok := auth.RequestHeaders["Authorization"].Matcher.(MatchCondition)
	require.True(t, ok, "Expected Authorization header matcher to be a MatchCondition")
	require.Equal(t, "Bearer", authMatcher.Value)
	require.Equal(t, "Contains", authMatcher.Operator)
	require.Equal(t, "Unauthorized", auth.Response.Content)
	require.Equal(t, 401, auth.Response.StatusCode)
	require.Equal(t, "Bearer realm=\"test\"", auth.Response.Headers["WWW-Authenticate"])
	require.False(t, auth.Continue)

	// Check second interceptor
	metrics := cfg.Interceptors[1]
	require.Equal(t, "/metrics", metrics.Path)
	require.Equal(t, "GET", metrics.Method)
	require.Equal(t, "Metrics collected", metrics.Response.Content)
	require.Equal(t, 200, metrics.Response.StatusCode)
	require.True(t, metrics.Continue)
}

func TestLoadConfig_WithSystem(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	imposterConfig := &ImposterConfig{}

	// Create a test data file
	dataContent := `{
		"key1": "value1",
		"key2": {
			"nested": "value2"
		}
	}`
	err := os.WriteFile(filepath.Join(tempDir, "data.json"), []byte(dataContent), 0644)
	require.NoError(t, err)

	// Create a test config file with system configuration
	configContent := `plugin: rest
system:
  stores:
    store1:
      preloadFile: data.json
    store2:
      preloadData:
        key3: value3
        key4:
          nested: value4
resources:
  - path: /test
    response:
      content: test response
      statusCode: 200`

	err = os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	configs := LoadConfig(tempDir, imposterConfig)
	require.Len(t, configs, 1)

	cfg := configs[0]
	require.Equal(t, "rest", cfg.Plugin)
	require.NotNil(t, cfg.System)
	require.Len(t, cfg.System.Stores, 2)

	// Check first store
	store1 := cfg.System.Stores["store1"]
	require.Equal(t, "data.json", store1.PreloadFile)
	require.Empty(t, store1.PreloadData)

	// Check second store
	store2 := cfg.System.Stores["store2"]
	require.Empty(t, store2.PreloadFile)
	require.NotNil(t, store2.PreloadData)
	require.Equal(t, "value3", store2.PreloadData["key3"])
	require.NotNil(t, store2.PreloadData["key4"])
	require.Equal(t, "value4", store2.PreloadData["key4"].(map[string]interface{})["nested"])
}

func TestLoadImposterConfig(t *testing.T) {
	// Test with environment variable set
	os.Setenv("IMPOSTER_PORT", "9090")
	cfg := LoadImposterConfig()
	require.Equal(t, "9090", cfg.ServerPort)

	// Test with environment variable unset
	os.Unsetenv("IMPOSTER_PORT")
	cfg = LoadImposterConfig()
	require.Equal(t, "8080", cfg.ServerPort)
}

func TestLoadConfig_WithRequestBody(t *testing.T) {
	imposterConfig := &ImposterConfig{}

	// Load the config from testdata
	configs := LoadConfig("testdata", imposterConfig)
	require.Len(t, configs, 1)

	cfg := configs[0]
	require.Equal(t, "rest", cfg.Plugin)
	require.Len(t, cfg.Resources, 4)

	// Test simple match
	simpleMatch := findResourceByPath(cfg.Resources, "/simple-match")
	require.NotNil(t, simpleMatch)
	require.Equal(t, "test content", simpleMatch.RequestBody.Value)
	require.Equal(t, "EqualTo", simpleMatch.RequestBody.Operator)

	// Test JSON match
	jsonMatch := findResourceByPath(cfg.Resources, "/json-match")
	require.NotNil(t, jsonMatch)
	require.Equal(t, "$.user.id", jsonMatch.RequestBody.JSONPath)
	require.Equal(t, "123", jsonMatch.RequestBody.Value)
	require.Equal(t, "EqualTo", jsonMatch.RequestBody.Operator)

	// Test XML match
	xmlMatch := findResourceByPath(cfg.Resources, "/xml-match")
	require.NotNil(t, xmlMatch)
	require.Equal(t, "//user/id", xmlMatch.RequestBody.XPath)
	require.Equal(t, "456", xmlMatch.RequestBody.Value)
	require.Equal(t, "Contains", xmlMatch.RequestBody.Operator)
	require.Equal(t, "http://example.com/ns1", xmlMatch.RequestBody.XMLNamespaces["ns1"])
	require.Equal(t, "http://example.com/ns2", xmlMatch.RequestBody.XMLNamespaces["ns2"])

	// Test multiple conditions
	multiMatch := findResourceByPath(cfg.Resources, "/multiple-conditions")
	require.NotNil(t, multiMatch)

	// Test allOf conditions
	require.Len(t, multiMatch.RequestBody.AllOf, 2)
	require.Equal(t, "$.type", multiMatch.RequestBody.AllOf[0].JSONPath)
	require.Equal(t, "user", multiMatch.RequestBody.AllOf[0].Value)
	require.Equal(t, "EqualTo", multiMatch.RequestBody.AllOf[0].Operator)
	require.Equal(t, "//status", multiMatch.RequestBody.AllOf[1].XPath)
	require.Equal(t, "active", multiMatch.RequestBody.AllOf[1].Value)
	require.Equal(t, "EqualTo", multiMatch.RequestBody.AllOf[1].Operator)

	// Test anyOf conditions
	require.Len(t, multiMatch.RequestBody.AnyOf, 2)
	require.Equal(t, "$.role", multiMatch.RequestBody.AnyOf[0].JSONPath)
	require.Equal(t, "admin", multiMatch.RequestBody.AnyOf[0].Value)
	require.Equal(t, "EqualTo", multiMatch.RequestBody.AnyOf[0].Operator)
	require.Equal(t, "$.permissions", multiMatch.RequestBody.AnyOf[1].JSONPath)
	require.Equal(t, "write", multiMatch.RequestBody.AnyOf[1].Value)
	require.Equal(t, "Contains", multiMatch.RequestBody.AnyOf[1].Operator)
}

// Helper function to find a resource by path
func findResourceByPath(resources []Resource, path string) *Resource {
	for _, r := range resources {
		if r.Path == path {
			return &r
		}
	}
	return nil
}

func TestLoadConfig_WithValidation(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	imposterConfig := &ImposterConfig{}

	// Create a test config file with validation configuration
	configContent := `plugin: openapi
specFile: petstore.yaml
validation:
  request: true
  response: false
resources:
  - path: /test
    response:
      content: test response
      statusCode: 200`

	err := os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	configs := LoadConfig(tempDir, imposterConfig)
	require.Len(t, configs, 1)

	cfg := configs[0]
	require.Equal(t, "openapi", cfg.Plugin)
	require.Equal(t, "petstore.yaml", cfg.SpecFile)

	// Check validation configuration
	require.NotNil(t, cfg.Validation)
	require.True(t, cfg.Validation.Request)
	require.False(t, cfg.Validation.Response)
}
