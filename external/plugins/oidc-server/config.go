package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type OIDCConfig struct {
	Users   []User   `yaml:"users"`
	Clients []Client `yaml:"clients"`
}

type User struct {
	Username string            `yaml:"username"`
	Password string            `yaml:"password"`
	Claims   map[string]string `yaml:"claims"`
}

type Client struct {
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	RedirectURIs []string `yaml:"redirect_uris"`
}

func loadOIDCConfig(configDir string) (*OIDCConfig, error) {
	configFile := filepath.Join(configDir, "oidc.yaml")

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Try alternative naming
		configFile = filepath.Join(configDir, "oidc-users.yaml")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			return getDefaultConfig(), nil
		}
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	var config OIDCConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func validateConfig(config *OIDCConfig) error {
	if len(config.Users) == 0 {
		return fmt.Errorf("at least one user must be configured")
	}

	if len(config.Clients) == 0 {
		return fmt.Errorf("at least one client must be configured")
	}

	// Validate users
	for i, user := range config.Users {
		if user.Username == "" {
			return fmt.Errorf("user at index %d: username is required", i)
		}
		if user.Password == "" {
			return fmt.Errorf("user at index %d: password is required", i)
		}
		if user.Claims == nil {
			user.Claims = make(map[string]string)
		}
		// Ensure sub claim exists
		if _, exists := user.Claims["sub"]; !exists {
			user.Claims["sub"] = user.Username
		}
	}

	// Validate clients
	for i, client := range config.Clients {
		if client.ClientID == "" {
			return fmt.Errorf("client at index %d: client_id is required", i)
		}
		if len(client.RedirectURIs) == 0 {
			return fmt.Errorf("client at index %d: at least one redirect_uri is required", i)
		}
	}

	return nil
}

func getDefaultConfig() *OIDCConfig {
	return &OIDCConfig{
		Users: []User{
			{
				Username: "alice",
				Password: "password",
				Claims: map[string]string{
					"sub":         "alice",
					"name":        "Alice Smith",
					"given_name":  "Alice",
					"family_name": "Smith",
					"email":       "alice@example.com",
				},
			},
			{
				Username: "bob",
				Password: "password",
				Claims: map[string]string{
					"sub":         "bob",
					"name":        "Bob Jones",
					"given_name":  "Bob",
					"family_name": "Jones",
					"email":       "bob@example.com",
				},
			},
		},
		Clients: []Client{
			{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				RedirectURIs: []string{
					"http://localhost:8080/callback",
					"http://localhost:3000/callback",
				},
			},
		},
	}
}

func (c *OIDCConfig) GetUser(username string) *User {
	for _, user := range c.Users {
		if user.Username == username {
			return &user
		}
	}
	return nil
}

func (c *OIDCConfig) GetClient(clientID string) *Client {
	for _, client := range c.Clients {
		if client.ClientID == clientID {
			return &client
		}
	}
	return nil
}

func (c *Client) IsValidRedirectURI(uri string) bool {
	for _, validURI := range c.RedirectURIs {
		if validURI == uri {
			return true
		}
	}
	return false
}
