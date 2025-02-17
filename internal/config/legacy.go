package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/imposter-project/imposter-go/pkg/logger"

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

// transformResponseConfig handles the transformation of response configuration
func transformResponseConfig(response *Response, rawResponse map[string]interface{}) ([]Step, error) {
	var steps []Step

	// Handle scriptFile to script step conversion
	if scriptFile, ok := rawResponse["scriptFile"].(string); ok {
		steps = []Step{
			{
				Type: "script",
				Lang: "javascript",
				File: scriptFile,
			},
		}
	}

	// Handle staticFile to file conversion
	if staticFile, ok := rawResponse["staticFile"].(string); ok {
		response.File = staticFile
	}

	// Copy other response fields if they exist
	if content, ok := rawResponse["content"].(string); ok {
		response.Content = content
	}
	if staticData, ok := rawResponse["staticData"].(string); ok {
		response.Content = staticData
	}
	if statusCode, ok := rawResponse["statusCode"].(int); ok {
		response.StatusCode = statusCode
	}
	if file, ok := rawResponse["file"].(string); ok {
		response.File = file
	}
	if headers, ok := rawResponse["headers"].(map[string]interface{}); ok {
		if response.Headers == nil {
			response.Headers = make(map[string]string)
		}
		for k, v := range headers {
			if strVal, ok := v.(string); ok {
				response.Headers[k] = strVal
			}
		}
	}

	return steps, nil
}

// transformLegacyConfig converts a legacy config format to the current format
func transformLegacyConfig(data []byte) ([]byte, error) {
	logger.Tracef("transforming legacy config format")

	// First unmarshal into a map to handle dynamic fields
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		logger.Debugf("failed to unmarshal raw config: %v", err)
		return nil, fmt.Errorf("failed to unmarshal raw config: %w", err)
	}

	// Create the current format config
	currentConfig := Config{
		Plugin: rawConfig["plugin"].(string),
	}

	// Handle root level request properties
	resource := Resource{}

	if path, ok := rawConfig["path"].(string); ok {
		resource.RequestMatcher = RequestMatcher{
			Path: path,
		}
	}
	if method, ok := rawConfig["method"].(string); ok {
		resource.Method = method
	}
	if contentType, ok := rawConfig["contentType"].(string); ok {
		if resource.Response.Headers == nil {
			resource.Response.Headers = make(map[string]string)
		}
		resource.Response.Headers["Content-Type"] = contentType
	}
	if response, ok := rawConfig["response"].(map[string]interface{}); ok {
		steps, err := transformResponseConfig(&resource.Response, response)
		if err != nil {
			return nil, err
		}
		if steps != nil {
			resource.Steps = steps
		}
	}
	currentConfig.Resources = []Resource{resource}

	// Handle resources array if it exists
	if resources, ok := rawConfig["resources"].([]interface{}); ok {
		currentConfig.Resources = make([]Resource, 0, len(resources))
		for _, res := range resources {
			resource := Resource{}
			resMap := res.(map[string]interface{})

			// Copy basic request matcher fields
			if path, ok := resMap["path"].(string); ok {
				resource.Path = path
			}
			if method, ok := resMap["method"].(string); ok {
				resource.Method = method
			}

			// Handle contentType at resource level
			if contentType, ok := resMap["contentType"].(string); ok {
				if resource.Response.Headers == nil {
					resource.Response.Headers = make(map[string]string)
				}
				resource.Response.Headers["Content-Type"] = contentType
			}

			// Handle response if present
			if response, ok := resMap["response"].(map[string]interface{}); ok {
				steps, err := transformResponseConfig(&resource.Response, response)
				if err != nil {
					return nil, err
				}
				if steps != nil {
					resource.Steps = steps
				}
			}

			currentConfig.Resources = append(currentConfig.Resources, resource)
		}
	}

	// Marshal back to YAML
	newData, err := yaml.Marshal(currentConfig)
	if err != nil {
		logger.Debugf("failed to marshal transformed config: %v", err)
		return nil, fmt.Errorf("failed to marshal transformed config: %w", err)
	}
	logger.Debugf("successfully transformed legacy config")
	logger.Tracef("transformed config: %s", string(newData))

	return newData, nil
}
