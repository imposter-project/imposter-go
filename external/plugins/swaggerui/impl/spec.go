package main

import (
	"encoding/json"
	"fmt"
	"github.com/imposter-project/imposter-go/external/handler"
	"strings"
)

var specConfigs []SpecConfig
var specConfigJSON string

type SpecConfig struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	OriginalPath string `json:"-"`
}

func generateSpecConfig(configs []handler.LightweightConfig) error {
	for _, cfg := range configs {
		if cfg.SpecFile == "" {
			continue
		}
		specFile := strings.TrimPrefix(cfg.SpecFile, "/")
		specConfigs = append(specConfigs, SpecConfig{
			Name:         specFile,
			OriginalPath: cfg.SpecFile,
			URL:          specPrefixPath + "/openapi/" + specFile,
		})
	}

	// serialise configs to JSON and return from the function
	jsonData, err := json.Marshal(specConfigs)
	if err != nil {
		return fmt.Errorf("failed to marshal spec config JSON: %w", err)
	}
	specConfigJSON = string(jsonData)
	return nil
}
