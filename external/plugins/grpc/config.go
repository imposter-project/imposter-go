package main

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// GRPCConfig holds the plugin configuration parsed from the config: block.
// Responses are defined in the standard resources: block, not here.
type GRPCConfig struct {
	ProtoFiles []string `yaml:"protoFiles"`
}

// loadGRPCConfig loads the gRPC plugin configuration from raw YAML bytes.
func loadGRPCConfig(pluginConfigBytes []byte) (*GRPCConfig, error) {
	if len(pluginConfigBytes) == 0 {
		return nil, fmt.Errorf("grpc plugin requires configuration with protoFiles")
	}

	var config GRPCConfig
	if err := yaml.Unmarshal(pluginConfigBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal grpc plugin config: %w", err)
	}

	if len(config.ProtoFiles) == 0 {
		return nil, fmt.Errorf("at least one proto file must be specified in protoFiles")
	}

	return &config, nil
}
