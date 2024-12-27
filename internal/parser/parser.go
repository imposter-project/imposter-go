package parser

import (
	"fmt"
	"github.com/gatehill/imposter-go/internal/config"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

// ParseConfig loads and parses a YAML configuration file
func ParseConfig(path string) (*config.Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &cfg, nil
}
