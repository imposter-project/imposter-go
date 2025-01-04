package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/imposter-project/imposter-go/internal/logger"
	"gopkg.in/yaml.v3"
)

// isLegacyConfig checks if the YAML data represents a legacy config format
func isLegacyConfig(data []byte) bool {
	// Check if legacy config support is enabled
	if strings.ToLower(os.Getenv("IMPOSTER_SUPPORT_LEGACY_CONFIG")) != "true" {
		logger.Tracef("legacy config support is disabled")
		return false
	}
	logger.Tracef("legacy config support is enabled")

	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return false
	}

	// Check if it has a direct response field at the root level
	_, hasResponse := rawConfig["response"]
	return hasResponse
}

// transformLegacyConfig converts a legacy config format to the current format
func transformLegacyConfig(data []byte) ([]byte, error) {
	logger.Debugf("transforming legacy config format")

	var legacyConfig struct {
		Plugin   string   `yaml:"plugin"`
		Response Response `yaml:"response"`
	}

	if err := yaml.Unmarshal(data, &legacyConfig); err != nil {
		logger.Debugf("failed to unmarshal legacy config: %v", err)
		return nil, fmt.Errorf("failed to unmarshal legacy config: %w", err)
	}
	logger.Tracef("unmarshalled legacy config with plugin: %s", legacyConfig.Plugin)

	// Handle the case where staticFile is used instead of file
	if legacyConfig.Response.File == "" {
		logger.Tracef("no file field found, checking for staticFile")
		var rawConfig map[string]interface{}
		if err := yaml.Unmarshal(data, &rawConfig); err != nil {
			logger.Debugf("failed to unmarshal raw config: %v", err)
			return nil, fmt.Errorf("failed to unmarshal raw config: %w", err)
		}
		if response, ok := rawConfig["response"].(map[string]interface{}); ok {
			if staticFile, ok := response["staticFile"].(string); ok {
				logger.Tracef("found staticFile: %s", staticFile)
				legacyConfig.Response.File = staticFile
			}
		}
	}

	// Transform to current format
	currentConfig := Config{
		Plugin: legacyConfig.Plugin,
		Resources: []Resource{
			{
				Response: legacyConfig.Response,
			},
		},
	}
	logger.Tracef("transformed to current format with plugin: %s", currentConfig.Plugin)

	// Marshal back to YAML
	newData, err := yaml.Marshal(currentConfig)
	if err != nil {
		logger.Debugf("failed to marshal transformed config: %v", err)
		return nil, fmt.Errorf("failed to marshal transformed config: %w", err)
	}
	logger.Debugf("successfully transformed legacy config")

	return newData, nil
}
