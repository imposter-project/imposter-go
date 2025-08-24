package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type OIDCConfig struct {
	Users     []User     `yaml:"users"`
	Clients   []Client   `yaml:"clients"`
	JWTConfig *JWTConfig `yaml:"jwt,omitempty"`
}

type JWTConfig struct {
	Algorithm     string `yaml:"algorithm"`   // "HS256" or "RS256"
	Secret        string `yaml:"secret"`      // HMAC secret for HS256
	PrivateKeyPEM string `yaml:"private_key"` // PEM encoded private key for RS256
	PublicKeyPEM  string `yaml:"public_key"`  // PEM encoded public key for RS256
	KeyID         string `yaml:"key_id"`      // Key ID for RS256
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

// loadOIDCConfig loads OIDC configuration from raw YAML bytes
// as provided by the main config system's plugin config block
func loadOIDCConfig(pluginConfigBytes []byte) (*OIDCConfig, error) {
	if len(pluginConfigBytes) == 0 {
		return getDefaultConfig(), nil
	}

	// Unmarshal directly from YAML bytes to strongly-typed struct
	var config OIDCConfig
	if err := yaml.Unmarshal(pluginConfigBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid plugin configuration: %w", err)
	}

	// Set default JWT config if not provided
	if config.JWTConfig == nil {
		config.JWTConfig = &JWTConfig{
			Algorithm: "RS256",
		}
	}

	// Validate JWT config
	if err := validateJWTConfig(config.JWTConfig); err != nil {
		return nil, fmt.Errorf("invalid JWT configuration: %w", err)
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

	// Set default JWT config if not provided
	if config.JWTConfig == nil {
		config.JWTConfig = &JWTConfig{
			Algorithm: "RS256",
		}
	}

	// Validate JWT config
	if err := validateJWTConfig(config.JWTConfig); err != nil {
		return fmt.Errorf("invalid JWT configuration: %w", err)
	}

	return nil
}

func validateJWTConfig(jwtConfig *JWTConfig) error {
	if jwtConfig.Algorithm == "" {
		jwtConfig.Algorithm = "RS256" // Default to RS256 (spec-compliant)
	}

	switch jwtConfig.Algorithm {
	case "HS256":
		// Just validate secret length if provided
		if jwtConfig.Secret != "" && len(jwtConfig.Secret) < 32 {
			return fmt.Errorf("HS256 secret must be at least 32 characters long for security")
		}
	case "RS256":
		// Just validate that if keys are provided, they're complete
		if (jwtConfig.PrivateKeyPEM != "" || jwtConfig.PublicKeyPEM != "") &&
			(jwtConfig.PrivateKeyPEM == "" || jwtConfig.PublicKeyPEM == "") {
			return fmt.Errorf("both private_key and public_key must be provided for RS256, or neither (for auto-generation)")
		}
	default:
		return fmt.Errorf("unsupported algorithm: %s (supported: HS256, RS256)", jwtConfig.Algorithm)
	}

	return nil
}

func generateHMACSecret() (string, error) {
	// Generate a 64-byte (512-bit) random secret
	bytes := make([]byte, 64)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func generateRSAKeyPair() (privateKeyPEM, publicKeyPEM, keyID string, err error) {
	// Generate RSA private key (2048 bits)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate RSA private key: %w", err)
	}

	// Convert private key to PKCS#8 format
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	// PEM encode private key
	privateKeyPEMBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privateKeyPEMBytes := pem.EncodeToMemory(privateKeyPEMBlock)

	// Convert public key to PKIX format
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	// PEM encode public key
	publicKeyPEMBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyPEMBytes := pem.EncodeToMemory(publicKeyPEMBlock)

	// Generate a unique key ID
	keyID = uuid.New().String()

	return string(privateKeyPEMBytes), string(publicKeyPEMBytes), keyID, nil
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
		JWTConfig: &JWTConfig{
			Algorithm: "RS256",
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
