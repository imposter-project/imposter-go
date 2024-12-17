package config

import (
	"os"
)

// Config holds application-wide configuration settings
type Config struct {
	ServerPort string
}

// LoadConfig loads configurations from environment variables
func LoadConfig() *Config {
	port := os.Getenv("IMPOSTER_PORT")
	if port == "" {
		port = "8080" // Default port
	}

	return &Config{
		ServerPort: port,
	}
}
