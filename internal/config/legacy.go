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

// isLegacyConfig checks if the YAML data represents a legacy config format
func isLegacyConfig(data []byte) bool {
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		return false
	}

	return hasLegacyFields(rawConfig)
}

// hasLegacyFields checks if a map contains any legacy configuration fields
func hasLegacyFields(config map[string]interface{}) bool {
	if hasLegacyRootFields(config) {
		return true
	}

	if resources, ok := config["resources"].([]interface{}); ok {
		for _, res := range resources {
			if resourceMap, ok := res.(map[string]interface{}); ok {
				if hasLegacyResourceFields(resourceMap) {
					return true
				}
			}
		}
	}

	return false
}

// hasLegacyRootFields checks if a map contains any legacy root level fields
func hasLegacyRootFields(config map[string]interface{}) bool {
	// Check for legacy root level fields
	_, hasPath := config["path"]
	_, hasMethod := config["method"]
	_, hasContentType := config["contentType"]
	_, hasResponse := config["response"]

	return hasPath || hasMethod || hasContentType || hasResponse
}

// hasLegacyResourceFields checks if a map contains any legacy resource level fields
func hasLegacyResourceFields(config map[string]interface{}) bool {
	_, hasContentType := config["contentType"]
	if hasContentType {
		return true
	}

	// Check for legacy response fields
	if response, ok := config["response"].(map[string]interface{}); ok {
		_, hasStaticFile := response["staticFile"]
		_, hasStaticData := response["staticData"]
		_, hasScriptFile := response["scriptFile"]
		if hasStaticFile || hasStaticData || hasScriptFile {
			return true
		}
	}

	return false
}

// mustMarshal marshals an interface to YAML bytes, panicking on error
func mustMarshal(v interface{}) []byte {
	data, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// transformResponseConfig handles the transformation of response configuration
func transformResponseConfig(response *Response, rawResponse map[string]interface{}) ([]Step, error) {
	var steps []Step

	// First unmarshal the raw response into the Response struct to preserve all current format fields
	if err := yaml.Unmarshal(mustMarshal(rawResponse), response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response config: %w", err)
	}

	// Then handle legacy-specific fields that need transformation
	if scriptFile, ok := rawResponse["scriptFile"].(string); ok {
		steps = []Step{
			{
				Type: "script",
				Lang: "javascript",
				File: scriptFile,
			},
		}
	}

	// Handle legacy staticFile field
	if staticFile, ok := rawResponse["staticFile"].(string); ok {
		response.File = staticFile
	}

	// Handle legacy staticData field
	if staticData, ok := rawResponse["staticData"].(string); ok {
		response.Content = staticData
	}

	// Ensure headers map exists
	if response.Headers == nil {
		response.Headers = make(map[string]string)
	}

	return steps, nil
}

// transformLegacyResponse is now just a wrapper around transformResponseConfig
func transformLegacyResponse(resource *Resource, rawResponse map[string]interface{}) {
	steps, _ := transformResponseConfig(&resource.Response, rawResponse)
	if steps != nil {
		resource.Steps = append(resource.Steps, steps...)
	}
}

// transformLegacyConfig converts a legacy config format to the current format
func transformLegacyConfig(data []byte) ([]byte, error) {
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
	if hasRootLegacyFields, resource := parseRootLegacyFields(rawConfig); hasRootLegacyFields {
		// Add as first resource if we found legacy fields
		if len(currentConfig.Resources) == 0 {
			currentConfig.Resources = []Resource{resource}
		} else {
			currentConfig.Resources = append([]Resource{resource}, currentConfig.Resources...)
		}
	}

	// Handle legacy fields in resources
	if resources, ok := rawConfig["resources"].([]interface{}); ok {
		for i, res := range resources {
			if resMap, ok := res.(map[string]interface{}); ok {
				// Only transform if legacy fields are present
				if hasLegacyResourceFields(resMap) {
					if i < len(currentConfig.Resources) {
						transformLegacyResource(&currentConfig.Resources[i], resMap)
					}
				}
			}
		}
	}

	// Marshal back to YAML
	newData, err := yaml.Marshal(currentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformed config: %w", err)
	}

	return newData, nil
}

// parseRootLegacyFields handles root-level legacy fields
func parseRootLegacyFields(rawConfig map[string]interface{}) (bool, Resource) {
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
		transformLegacyResponse(&resource, response)
	}
	return hasRootLegacyFields, resource
}

// transformLegacyResource handles legacy resource fields
func transformLegacyResource(resource *Resource, rawResource map[string]interface{}) {
	// Handle legacy contentType
	if contentType, ok := rawResource["contentType"].(string); ok {
		if resource.Response.Headers == nil {
			resource.Response.Headers = make(map[string]string)
		}
		resource.Response.Headers["Content-Type"] = contentType
	}

	// Handle legacy response
	if response, ok := rawResource["response"].(map[string]interface{}); ok {
		transformLegacyResponse(resource, response)
	}
}
