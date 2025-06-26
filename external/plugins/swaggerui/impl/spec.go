package main

import (
	"github.com/imposter-project/imposter-go/external/handler"
	"strings"
)

var specConfigs []SpecConfig

type SpecConfig struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	OriginalPath string `json:"-"`
	ConfigDir    string `json:"-"`
}

func generateSpecConfig(configs []handler.LightweightConfig) error {
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
func serveRawSpec(path string) *handler.HandlerResponse {
	var response *handler.HandlerResponse
	for _, specConfig := range specConfigs {
		if path == specConfig.URL {
			response = &handler.HandlerResponse{
				ConfigDir:  specConfig.ConfigDir,
				StatusCode: 200,
				File:       specConfig.OriginalPath,
			}
			break
		}
	}
	return response
}
