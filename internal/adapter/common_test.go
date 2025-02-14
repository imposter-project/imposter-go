package adapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialiseImposter(t *testing.T) {
	// Create a temporary directory for test configs
	tmpDir := t.TempDir()

	// Create a valid config file
	validConfig := []byte(`plugin: rest
basePath: /api
resources:
  - path: /test
    method: GET
    response:
      content: test response`)
	err := os.WriteFile(filepath.Join(tmpDir, "rest-config.yml"), validConfig, 0644)
	require.NoError(t, err)

	// Create a store config file
	storeConfig := []byte(`plugin: rest
basePath: /api
system:
  stores:
    testStore:
      preloadData:
        key1: value1`)
	err = os.WriteFile(filepath.Join(tmpDir, "store-config.yml"), storeConfig, 0644)
	require.NoError(t, err)

	tests := []struct {
		name          string
		configDirArg  string
		envConfigDir  string
		envPort       string
		wantPanic     bool
		panicContains string
	}{
		{
			name:         "config dir from argument",
			configDirArg: tmpDir,
			envPort:      "8080",
		},
		{
			name:         "config dir from environment",
			envConfigDir: tmpDir,
			envPort:      "9090",
		},
		{
			name:          "missing config dir",
			wantPanic:     true,
			panicContains: "Config directory path must be provided either as an argument or via IMPOSTER_CONFIG_DIR environment variable",
		},
		{
			name:          "invalid config dir",
			configDirArg:  "/nonexistent/path",
			wantPanic:     true,
			panicContains: "Specified path is not a valid directory",
		},
		{
			name:         "default port",
			configDirArg: tmpDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.envConfigDir != "" {
				t.Setenv("IMPOSTER_CONFIG_DIR", tt.envConfigDir)
			} else {
				t.Setenv("IMPOSTER_CONFIG_DIR", "")
			}
			if tt.envPort != "" {
				t.Setenv("IMPOSTER_PORT", tt.envPort)
			} else {
				t.Setenv("IMPOSTER_PORT", "")
			}

			if tt.wantPanic {
				assert.PanicsWithValue(t, tt.panicContains, func() {
					InitialiseImposter(tt.configDirArg)
				})
				return
			}

			// Run initialisation
			imposterConfig, plugins := InitialiseImposter(tt.configDirArg)
			assert.NotEmpty(t, plugins)

			configs := []*config.Config{}
			for _, plugin := range plugins {
				configs = append(configs, plugin.GetConfig())
			}

			// Verify results
			assert.NotNil(t, imposterConfig)
			if tt.envPort != "" {
				assert.Equal(t, tt.envPort, imposterConfig.ServerPort)
			} else {
				assert.Equal(t, "8080", imposterConfig.ServerPort) // Default port
			}

			assert.NotEmpty(t, configs)
			assert.Len(t, configs, 2) // Our two test config files

			// Verify config contents
			var foundRest, foundStore bool
			for _, cfg := range configs {
				if cfg.Plugin == "rest" {
					if cfg.System != nil && cfg.System.Stores != nil {
						foundStore = true
						assert.Contains(t, cfg.System.Stores, "testStore")
						assert.Contains(t, cfg.System.Stores["testStore"].PreloadData, "key1")
						assert.Equal(t, "value1", cfg.System.Stores["testStore"].PreloadData["key1"])
					} else {
						foundRest = true
						assert.Equal(t, "/api", cfg.BasePath)
						assert.Len(t, cfg.Resources, 1)
						assert.Equal(t, "/api/test", cfg.Resources[0].Path)
						assert.Equal(t, "GET", cfg.Resources[0].Method)
						assert.Equal(t, "test response", cfg.Resources[0].Response.Content)
					}
				}
			}
			assert.True(t, foundRest, "REST config not found")
			assert.True(t, foundStore, "Store config not found")
		})
	}
}

func TestInitialiseImposter_StoreInitialisation(t *testing.T) {
	// Create a temporary directory for test configs
	tmpDir := t.TempDir()

	// Create a store config file with preload data
	storeConfig := []byte(`plugin: rest
basePath: /api
system:
  stores:
    testStore:
      preloadData:
        key1: value1
        key2: value2`)
	err := os.WriteFile(filepath.Join(tmpDir, "store-config.yml"), storeConfig, 0644)
	require.NoError(t, err)

	// Run initialisation
	_, _ = InitialiseImposter(tmpDir)

	// Verify store initialisation
	// Note: Since store is a global singleton, we can verify its state here
	stores := []string{"testStore"}
	for _, storeName := range stores {
		// Verify preloaded data
		s := store.Open(storeName, nil)
		value, exists := s.GetValue("key1")
		require.True(t, exists)
		assert.Equal(t, "value1", value)

		value, exists = s.GetValue("key2")
		require.True(t, exists)
		assert.Equal(t, "value2", value)
	}
}

func TestInitialiseImposter_ConfigValidation(t *testing.T) {
	// Create a temporary directory for test configs
	tmpDir := t.TempDir()

	// Create an invalid config file (missing required fields)
	invalidConfig := []byte(`plugin: rest
# Missing basePath
resources:
  - # Missing path and method
    response:
      content: test response`)
	err := os.WriteFile(filepath.Join(tmpDir, "invalid-config.yml"), invalidConfig, 0644)
	require.NoError(t, err)

	// Run initialisation - it should not panic on invalid config
	imposterConfig, plugins := InitialiseImposter(tmpDir)
	assert.NotEmpty(t, plugins)

	configs := []*config.Config{}
	for _, plugin := range plugins {
		configs = append(configs, plugin.GetConfig())
	}

	// Verify results
	assert.NotNil(t, imposterConfig)
	assert.NotEmpty(t, configs)
	// The invalid config should still be loaded, but might be partially populated
	assert.Len(t, configs, 1)
	if len(configs) > 0 {
		assert.Equal(t, "rest", configs[0].Plugin)
	}
}
