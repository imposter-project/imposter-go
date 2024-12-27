package config

import (
	"os"
)

// Response represents an HTTP response
type Response struct {
	Content    string `yaml:"content"`
	StatusCode int    `yaml:"statusCode"`
	File       string `yaml:"file"`
}

// Resource represents an HTTP resource
type Resource struct {
	Method      string            `yaml:"method"`
	Path        string            `yaml:"path"`
	QueryParams map[string]string `yaml:"queryParams"`
	Headers     map[string]string `yaml:"headers"`
	RequestBody map[string]string `yaml:"requestBody"`
	Response    Response          `yaml:"response"`
}

type Config struct {
	Plugin    string `yaml:"plugin"`
	Resources []Resource
}

// Application-wide configuration
type ImposterConfig struct {
	ServerPort string
}

// LoadConfig loads configurations from environment variables
func LoadConfig() *ImposterConfig {
	port := os.Getenv("IMPOSTER_PORT")
	if port == "" {
		port = "8080" // Default port
	}

	return &ImposterConfig{
		ServerPort: port,
	}
}
