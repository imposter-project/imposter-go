package parser

import (
	"fmt"
	"io/ioutil"
	"github.com/gatehill/imposter-go/internal/handler"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Plugin    string             `yaml:"plugin"`
	Resources []handler.Resource `yaml:"resources"`
}

// ParseConfig loads and parses a YAML configuration file
func ParseConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &config, nil
}