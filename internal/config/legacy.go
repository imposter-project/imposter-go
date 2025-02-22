package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/imposter-project/imposter-go/pkg/logger"

	"gopkg.in/yaml.v3"
)

// isLegacyConfigEnabled returns whether legacy config support is enabled
func isLegacyConfigEnabled() bool {
	legacySupport := strings.ToLower(os.Getenv("IMPOSTER_SUPPORT_LEGACY_CONFIG")) == "true"
	if legacySupport {
		logger.Debugln("legacy config support is enabled")
	} else {
		logger.Traceln("legacy config support is disabled")
	}
	return legacySupport
}

// transformLegacyConfig converts a legacy config format to the current format
func transformLegacyConfig(data []byte) (*Config, error) {
	logger.Tracef("transforming legacy config format")

	// First unmarshal into the main Config struct to capture all current fields
	var currentConfig Config
	if err := yaml.Unmarshal(data, &currentConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal as current config: %v", err)
	}

	// Then unmarshal into a map for legacy field handling
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw config: %w", err)
	}

	// Handle legacy root fields
	legacyRootResource, err := parseRootLegacyFields(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root legacy fields: %w", err)
	}
	if legacyRootResource != nil {
		currentConfig.Resources = append(currentConfig.Resources, *legacyRootResource)
	}

	// Handle legacy fields in resources
	if resources, ok := rawConfig["resources"].([]interface{}); ok {
		for i, res := range resources {
			if resMap, ok := res.(map[string]interface{}); ok {
				// Check for legacy fields
				if _, hasContentType := resMap["contentType"]; hasContentType {
					if i < len(currentConfig.Resources) {
						if err := transformLegacyResource(&currentConfig.Resources[i], resMap); err != nil {
							return nil, fmt.Errorf("failed to transform legacy resource %d: %w", i, err)
						}
					}
				} else if response, ok := resMap["response"].(map[string]interface{}); ok {
					if _, hasStaticFile := response["staticFile"]; hasStaticFile {
						if i < len(currentConfig.Resources) {
							if err := transformLegacyResource(&currentConfig.Resources[i], resMap); err != nil {
								return nil, fmt.Errorf("failed to transform legacy resource %d: %w", i, err)
							}
						}
					} else if _, hasStaticData := response["staticData"]; hasStaticData {
						if i < len(currentConfig.Resources) {
							if err := transformLegacyResource(&currentConfig.Resources[i], resMap); err != nil {
								return nil, fmt.Errorf("failed to transform legacy resource %d: %w", i, err)
							}
						}
					} else if _, hasScriptFile := response["scriptFile"]; hasScriptFile {
						if i < len(currentConfig.Resources) {
							if err := transformLegacyResource(&currentConfig.Resources[i], resMap); err != nil {
								return nil, fmt.Errorf("failed to transform legacy resource %d: %w", i, err)
							}
						}
					}
				}
			}
		}
	}

	return &currentConfig, nil
}

// parseRootLegacyFields handles root-level legacy fields
func parseRootLegacyFields(rawConfig map[string]interface{}) (*Resource, error) {
	var hasRootLegacyFields bool
	resource := Resource{
		RequestMatcher: RequestMatcher{},
	}

	if path, ok := rawConfig["path"].(string); ok {
		hasRootLegacyFields = true
		resource.Path = path
	}

	// Copy other root-level legacy fields
	if method, ok := rawConfig["method"].(string); ok {
		hasRootLegacyFields = true
		resource.Method = method
	}
	if contentType, ok := rawConfig["contentType"].(string); ok {
		hasRootLegacyFields = true
		if resource.Response.Headers == nil {
			resource.Response.Headers = make(map[string]string)
		}
		resource.Response.Headers["Content-Type"] = contentType
	}
	if response, ok := rawConfig["response"].(map[string]interface{}); ok {
		hasRootLegacyFields = true
		if err := transformResponseConfig(&resource, response); err != nil {
			return nil, fmt.Errorf("failed to transform root response: %w", err)
		}
	}
	if hasRootLegacyFields {
		return &resource, nil
	}
	return nil, nil
}

// transformLegacyResource handles legacy resource fields
func transformLegacyResource(resource *Resource, rawResource map[string]interface{}) error {
	// Handle legacy contentType
	if contentType, ok := rawResource["contentType"].(string); ok {
		if resource.Response.Headers == nil {
			resource.Response.Headers = make(map[string]string)
		}
		resource.Response.Headers["Content-Type"] = contentType
	}

	// Handle legacy response
	if response, ok := rawResource["response"].(map[string]interface{}); ok {
		if err := transformResponseConfig(resource, response); err != nil {
			return fmt.Errorf("failed to transform response: %w", err)
		}
	}
	return nil
}

// transformResponseConfig handles the transformation of response configuration
func transformResponseConfig(resource *Resource, rawResponse map[string]interface{}) error {
	// First unmarshal the raw response into the Response struct to preserve all current format fields
	if err := yaml.Unmarshal(mustMarshal(rawResponse), &resource.Response); err != nil {
		return fmt.Errorf("failed to unmarshal response config: %w", err)
	}

	// Then handle legacy-specific fields that need transformation
	if scriptFile, ok := rawResponse["scriptFile"].(string); ok {
		resource.Steps = append(resource.Steps, Step{
			Type: "script",
			Lang: "javascript",
			File: scriptFile,
		})
	}

	// Handle legacy staticFile field
	if staticFile, ok := rawResponse["staticFile"].(string); ok {
		resource.Response.File = staticFile
	}

	// Handle legacy staticData field
	if staticData, ok := rawResponse["staticData"].(string); ok {
		resource.Response.Content = staticData
	}

	// Ensure headers map exists
	if resource.Response.Headers == nil {
		resource.Response.Headers = make(map[string]string)
	}

	return nil
}

// mustMarshal marshals an interface to YAML bytes, panicking on error
func mustMarshal(v interface{}) []byte {
	data, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
