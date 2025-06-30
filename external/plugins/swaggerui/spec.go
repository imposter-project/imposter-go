package main

import (
	"github.com/imposter-project/imposter-go/external/shared"
	"strings"
)

var specConfigs []SpecConfig

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

// serveRawSpec serves the raw OpenAPI spec file based on the provided path.
// If no matching spec is found, it returns nil.
func serveRawSpec(path string) *shared.HandlerResponse {
	var response *shared.HandlerResponse
	for _, specConfig := range specConfigs {
		if path == specConfig.URL {
			// TODO instead of serving the raw spec, parse it and add the server URL as the first server entry, then marshal it back to JSON.
			response = &shared.HandlerResponse{
				FileBaseDir: specConfig.ConfigDir,
				StatusCode:  200,
				File:        specConfig.OriginalPath,
			}
			break
		}
	}
	return response
}
