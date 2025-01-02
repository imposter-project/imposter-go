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
    operation:
      name: testOperation
      soapAction: testAction
    response:
      content: test response
      statusCode: 200
  - path: /another
    operation:
      name: anotherOperation
    response:
      content: another response
      statusCode: 200`

	err = os.WriteFile(filepath.Join(tempDir, "test-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	configs := LoadConfig(tempDir)
	require.Len(t, configs, 1)

	cfg := configs[0]
	require.Equal(t, "soap", cfg.Plugin)
	require.Equal(t, "test.wsdl", cfg.WSDLFile)
	require.Len(t, cfg.Resources, 2)

	// Check first resource
	require.Equal(t, "/test", cfg.Resources[0].Path)
	require.NotNil(t, cfg.Resources[0].Operation)
	require.Equal(t, "testOperation", cfg.Resources[0].Operation.Name)
	require.Equal(t, "testAction", cfg.Resources[0].Operation.SOAPAction)
	require.Equal(t, "test response", cfg.Resources[0].Response.Content)
	require.Equal(t, 200, cfg.Resources[0].Response.StatusCode)

	// Check second resource
	require.Equal(t, "/another", cfg.Resources[1].Path)
	require.NotNil(t, cfg.Resources[1].Operation)
	require.Equal(t, "anotherOperation", cfg.Resources[1].Operation.Name)
	require.Equal(t, "another response", cfg.Resources[1].Response.Content)
	require.Equal(t, 200, cfg.Resources[1].Response.StatusCode)
}

// ... rest of the file unchanged ...
