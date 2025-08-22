package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
)

func TestOIDCServer_RS256Support(t *testing.T) {
	// Generate test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}

	// Encode private key as PEM
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Encode public key as PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to marshal public key: %v", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Create OIDC config with RS256
	config := &OIDCConfig{
		Users: []User{
			{
				Username: "testuser",
				Password: "testpass",
				Claims: map[string]string{
					"sub":   "testuser",
					"name":  "Test User",
					"email": "test@example.com",
				},
			},
		},
		Clients: []Client{
			{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				RedirectURIs: []string{"http://localhost:8080/callback"},
			},
		},
		JWTConfig: &JWTConfig{
			Algorithm:     "RS256",
			PrivateKeyPEM: string(privateKeyPEM),
			PublicKeyPEM:  string(publicKeyPEM),
			KeyID:         "test-key-1",
		},
	}

	// Create OIDC server
	server := &OIDCServer{
		logger:    hclog.NewNullLogger(),
		config:    config,
		serverURL: "http://localhost:8080",
		sessions:  make(map[string]*AuthSession),
		codes:     make(map[string]*AuthCode),
		tokens:    make(map[string]*AccessToken),
	}

	// Setup JWT keys
	err = server.setupJWTKeys()
	if err != nil {
		t.Fatalf("Failed to setup JWT keys: %v", err)
	}

	// Cache discovery document
	err = server.CacheDiscoveryDocument()
	if err != nil {
		t.Fatalf("Failed to cache discovery document: %v", err)
	}

	// Test discovery endpoint shows RS256 support
	discoveryResp := server.handleDiscovery(shared.HandlerRequest{
		Method: "GET",
		Path:   "/.well-known/openid-configuration",
	})

	if discoveryResp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", discoveryResp.StatusCode)
	}

	discoveryBody := string(discoveryResp.Body)
	t.Logf("Discovery document: %s", discoveryBody)
	if !strings.Contains(discoveryBody, `"RS256"`) {
		t.Error("Discovery document should advertise RS256 support")
	}
	if !strings.Contains(discoveryBody, "jwks_uri") {
		t.Error("Discovery document should include jwks_uri")
	}

	// Test JWKS endpoint
	jwksResp := server.handleJWKS(shared.HandlerRequest{
		Method: "GET",
		Path:   "/.well-known/jwks.json",
	})

	if jwksResp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", jwksResp.StatusCode)
	}

	jwksBody := string(jwksResp.Body)
	if !strings.Contains(jwksBody, `"kty":"RSA"`) {
		t.Error("JWKS should contain RSA key")
	}
	if !strings.Contains(jwksBody, `"alg":"RS256"`) {
		t.Error("JWKS should specify RS256 algorithm")
	}
	if !strings.Contains(jwksBody, `"kid":"test-key-1"`) {
		t.Error("JWKS should include key ID")
	}

	// Test JWT generation uses RS256
	tokenString, err := server.generateIDToken(
		&config.Users[0],
		"test-client",
		"test-nonce",
		[]string{"openid", "profile"},
		// Use current time so token doesn't expire during test
		time.Now(),
		3600,
	)
	if err != nil {
		t.Fatalf("Failed to generate ID token: %v", err)
	}

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			t.Errorf("Expected RS256 signing method, got %v", token.Header["alg"])
		}

		// Verify key ID in header
		if token.Header["kid"] != "test-key-1" {
			t.Errorf("Expected kid 'test-key-1', got %v", token.Header["kid"])
		}

		return &privateKey.PublicKey, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse/verify token: %v", err)
	}

	if !token.Valid {
		t.Error("Token should be valid")
	}
}

func TestOIDCServer_JWKSCaching(t *testing.T) {
	// Generate test RSA key pair
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	privateKeyBytes, _ := x509.MarshalPKCS8PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes})

	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})

	config := &OIDCConfig{
		Users:   []User{{Username: "test", Password: "test", Claims: map[string]string{"sub": "test"}}},
		Clients: []Client{{ClientID: "test", RedirectURIs: []string{"http://localhost:8080/callback"}}},
		JWTConfig: &JWTConfig{
			Algorithm:     "RS256",
			PrivateKeyPEM: string(privateKeyPEM),
			PublicKeyPEM:  string(publicKeyPEM),
			KeyID:         "test-key",
		},
	}

	server := &OIDCServer{
		logger:    hclog.NewNullLogger(),
		config:    config,
		serverURL: "http://localhost:8080",
	}

	// Setup JWT keys (this should cache the JWKS)
	err := server.setupJWTKeys()
	if err != nil {
		t.Fatalf("Failed to setup JWT keys: %v", err)
	}

	// Verify JWKS was cached
	if len(server.cachedJWKS) == 0 {
		t.Error("JWKS should be cached after setup")
	}

	// Call JWKS endpoint multiple times and verify we get the same cached response
	resp1 := server.handleJWKS(shared.HandlerRequest{Method: "GET", Path: "/.well-known/jwks.json"})
	resp2 := server.handleJWKS(shared.HandlerRequest{Method: "GET", Path: "/.well-known/jwks.json"})

	if resp1.StatusCode != 200 || resp2.StatusCode != 200 {
		t.Error("JWKS endpoint should return 200")
	}

	if string(resp1.Body) != string(resp2.Body) {
		t.Error("JWKS responses should be identical (cached)")
	}

	// Verify the response matches our cached data
	if string(resp1.Body) != string(server.cachedJWKS) {
		t.Error("JWKS response should match cached data")
	}
}

func TestOIDCServer_HS256Fallback(t *testing.T) {
	// Test that HS256 still works when no JWT config is provided
	config := getDefaultConfig() // This should default to HS256

	server := &OIDCServer{
		logger:    hclog.NewNullLogger(),
		config:    config,
		serverURL: "http://localhost:8080",
	}

	// Setup JWT keys
	err := server.setupJWTKeys()
	if err != nil {
		t.Fatalf("Failed to setup JWT keys: %v", err)
	}

	// Cache discovery document
	err = server.CacheDiscoveryDocument()
	if err != nil {
		t.Fatalf("Failed to cache discovery document: %v", err)
	}

	// Test discovery endpoint shows HS256 support
	discoveryResp := server.handleDiscovery(shared.HandlerRequest{
		Method: "GET",
		Path:   "/.well-known/openid-configuration",
	})

	discoveryBody := string(discoveryResp.Body)
	if !strings.Contains(discoveryBody, `"HS256"`) {
		t.Error("Discovery document should advertise HS256 support")
	}
	if strings.Contains(discoveryBody, `"RS256"`) {
		t.Error("Discovery document should not advertise RS256 for HS256 config")
	}

	// Test JWKS endpoint returns empty keys for HS256
	jwksResp := server.handleJWKS(shared.HandlerRequest{
		Method: "GET",
		Path:   "/.well-known/jwks.json",
	})

	jwksBody := string(jwksResp.Body)
	if jwksBody != `{"keys": []}` {
		t.Errorf("JWKS should return empty keys for HS256, got: %s", jwksBody)
	}
}

func TestOIDCServer_HS256WithSecret(t *testing.T) {
	// Test HS256 with configured secret
	configuredSecret := "my-secure-test-secret-that-is-long-enough-for-testing"

	config := &OIDCConfig{
		Users:   []User{{Username: "test", Password: "test", Claims: map[string]string{"sub": "test"}}},
		Clients: []Client{{ClientID: "test", RedirectURIs: []string{"http://localhost:8080/callback"}}},
		JWTConfig: &JWTConfig{
			Algorithm: "HS256",
			Secret:    configuredSecret,
		},
	}

	server := &OIDCServer{
		logger:    hclog.NewNullLogger(),
		config:    config,
		serverURL: "http://localhost:8080",
	}

	// Setup JWT keys
	err := server.setupJWTKeys()
	if err != nil {
		t.Fatalf("Failed to setup JWT keys: %v", err)
	}

	// Verify the secret was used
	if string(server.jwtSecret) != configuredSecret {
		t.Errorf("Expected JWT secret to be configured secret, got: %s", string(server.jwtSecret))
	}

	// Test JWT generation with configured secret
	tokenString, err := server.generateIDToken(
		&config.Users[0],
		"test-client",
		"test-nonce",
		[]string{"openid", "profile"},
		time.Now(),
		3600,
	)
	if err != nil {
		t.Fatalf("Failed to generate ID token: %v", err)
	}

	// Verify the token can be parsed with the configured secret
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(configuredSecret), nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !token.Valid {
		t.Error("Token should be valid")
	}
}

func TestValidateJWTConfig_SecretValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *JWTConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid HS256 with long secret",
			config: &JWTConfig{
				Algorithm: "HS256",
				Secret:    "this-is-a-very-secure-secret-that-is-definitely-long-enough",
			},
			expectError: false,
		},
		{
			name: "HS256 with short secret",
			config: &JWTConfig{
				Algorithm: "HS256",
				Secret:    "short",
			},
			expectError: true,
			errorMsg:    "HS256 secret must be at least 32 characters long",
		},
		{
			name: "HS256 without secret",
			config: &JWTConfig{
				Algorithm: "HS256",
			},
			expectError: false, // Should not error, will generate random secret with warning
		},
		{
			name: "HS256 with exactly 32 character secret",
			config: &JWTConfig{
				Algorithm: "HS256",
				Secret:    "exactly-32-characters-long-key!!",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWTConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
