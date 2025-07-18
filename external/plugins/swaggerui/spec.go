package main

import (
	"encoding/json"
	"fmt"
	"github.com/imposter-project/imposter-go/external/shared"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var specConfigs []SpecConfig

var mu sync.RWMutex
var processedSpecs = make(map[string][]byte)

type SpecConfig struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	OriginalPath string `json:"-"`
	ConfigDir    string `json:"-"`
}

func generateSpecConfig(configs []shared.LightweightConfig) error {
	for _, cfg := range configs {
		if cfg.SpecFile == "" {
			continue
		}
		specFile := strings.TrimPrefix(cfg.SpecFile, "/")
		specConfigs = append(specConfigs, SpecConfig{
			Name:         specFile,
			URL:          specPrefixPath + "/openapi/" + specFile,
			OriginalPath: cfg.SpecFile,
			ConfigDir:    cfg.ConfigDir,
		})
	}
	return nil
}

// getServerURL constructs the server URL from environment variables
func getServerURL() string {
	serverURL := os.Getenv("IMPOSTER_SERVER_URL")
	if serverURL != "" {
		return serverURL
	}

	port := os.Getenv("IMPOSTER_PORT")
	if port == "" {
		port = "8080"
	}

	var hostSuffix string
	if port != "80" {
		hostSuffix = fmt.Sprintf(":%s", port)
	}
	return fmt.Sprintf("http://localhost%s", hostSuffix)
}

// serveRawSpec serves the OpenAPI spec file with server URL modifications.
// If no matching spec is found, it returns nil.
func serveRawSpec(path string) *shared.HandlerResponse {
	for _, specConfig := range specConfigs {
		if path == specConfig.URL {
			return getSpec(specConfig)
		}
	}
	return nil
}

func getSpec(specConfig SpecConfig) *shared.HandlerResponse {
	specPath := filepath.Join(specConfig.ConfigDir, specConfig.OriginalPath)
	jsonData := readSpecFromCache(specPath)

	if jsonData == nil {
		mu.Lock()
		defer mu.Unlock()

		logger.Trace("loading spec file", "path", specPath)
		specData, err := os.ReadFile(specPath)
		if err != nil {
			return &shared.HandlerResponse{
				StatusCode: 500,
				Body:       []byte(fmt.Sprintf("Error reading spec file: %v", err)),
			}
		}

		// YAML parser will handle both YAML and JSON formats
		var specMap map[string]interface{}
		if err := yaml.Unmarshal(specData, &specMap); err != nil {
			return &shared.HandlerResponse{
				StatusCode: 500,
				Body:       []byte(fmt.Sprintf("Error parsing spec file: %v", err)),
			}
		}

		serverURL := getServerURL()
		appendServerUrl(specMap, serverURL)

		jsonData, err = json.MarshalIndent(specMap, "", "  ")
		if err != nil {
			return &shared.HandlerResponse{
				StatusCode: 500,
				Body:       []byte(fmt.Sprintf("Error marshalling spec: %v", err)),
			}
		}

		processedSpecs[specPath] = jsonData
	}

	return &shared.HandlerResponse{
		StatusCode: 200,
		Body:       jsonData,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

func readSpecFromCache(specPath string) []byte {
	mu.RLock()
	defer mu.RUnlock()
	return processedSpecs[specPath]
}

// appendServerUrl modifies the OpenAPI spec map to include the server URL.
func appendServerUrl(specMap map[string]interface{}, serverURL string) {
	// Check if this is OpenAPI 3.x or Swagger 2.0
	if openapi, exists := specMap["openapi"]; exists {
		// OpenAPI 3.x - add to servers array
		if openapiVersion, ok := openapi.(string); ok && strings.HasPrefix(openapiVersion, "3.") {
			servers, exists := specMap["servers"]
			if !exists {
				servers = []interface{}{}
			}

			serverList, ok := servers.([]interface{})
			if !ok {
				serverList = []interface{}{}
			}

			// Add server URL as first entry
			newServer := map[string]interface{}{"url": serverURL}
			serverList = append([]interface{}{newServer}, serverList...)
			specMap["servers"] = serverList
		}
	} else if _, exists := specMap["swagger"]; exists {
		// Swagger 2.0 - set basePath and host
		if swaggerVersion, ok := specMap["swagger"].(string); ok && strings.HasPrefix(swaggerVersion, "2.") {
			// Parse the server URL to extract host and basePath
			if strings.HasPrefix(serverURL, "http://") {
				serverURL = strings.TrimPrefix(serverURL, "http://")
			} else if strings.HasPrefix(serverURL, "https://") {
				serverURL = strings.TrimPrefix(serverURL, "https://")
				specMap["schemes"] = []interface{}{"https"}
			}

			parts := strings.SplitN(serverURL, "/", 2)
			specMap["host"] = parts[0]

			if len(parts) > 1 {
				specMap["basePath"] = "/" + parts[1]
			} else {
				specMap["basePath"] = "/"
			}
		}
	}
}
