package main

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
)

func createTestOIDCServerForPlugin() *OIDCServer {
	return &OIDCServer{
		logger: hclog.New(&hclog.LoggerOptions{
			Level:  hclog.Off, // Disable logging during tests
			Output: nil,
		}),
		config:    getDefaultConfig(),
		serverURL: "http://localhost:8080",
		sessions:  make(map[string]*AuthSession),
		codes:     make(map[string]*AuthCode),
		tokens:    make(map[string]*AccessToken),
		jwtSecret: []byte("test-secret-key-32-bytes-long!"),
	}
}

func TestOIDCServer_Configure(t *testing.T) {
	tests := []struct {
		name        string
		config      shared.ExternalConfig
		expectError bool
	}{
		{
			name: "valid config with server URL",
			config: shared.ExternalConfig{
				Server: shared.ServerConfig{
					URL: "https://example.com",
				},
				Configs: []shared.LightweightConfig{
					{ConfigDir: "/tmp"},
				},
			},
			expectError: false,
		},
		{
			name: "empty server URL uses fallback",
			config: shared.ExternalConfig{
				Server:  shared.ServerConfig{URL: ""},
				Configs: []shared.LightweightConfig{},
			},
			expectError: false,
		},
		{
			name: "no configs uses default",
			config: shared.ExternalConfig{
				Server:  shared.ServerConfig{URL: "https://test.com"},
				Configs: []shared.LightweightConfig{},
			},
			expectError: false,
		},
		{
			name: "config with plugin config block",
			config: shared.ExternalConfig{
				Server: shared.ServerConfig{URL: "https://plugin-test.com"},
				Configs: []shared.LightweightConfig{
					{
						ConfigDir: "/tmp",
						PluginConfig: []byte(`
users:
  - username: "pluginuser"
    password: "pluginpass"
    claims:
      sub: "pluginuser"
      email: "plugin@test.com"
clients:
  - client_id: "pluginclient"
    client_secret: "pluginsecret"
    redirect_uris:
      - "https://plugin-test.com/callback"
`),
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &OIDCServer{
				logger: hclog.New(&hclog.LoggerOptions{
					Level:  hclog.Off,
					Output: nil,
				}),
			}

			err := server.Configure(tt.config)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				// Check that server was properly configured
				if server.sessions == nil {
					t.Error("Expected sessions map to be initialized")
				}
				if server.codes == nil {
					t.Error("Expected codes map to be initialized")
				}
				if server.tokens == nil {
					t.Error("Expected tokens map to be initialized")
				}
				if server.jwtSecret == nil {
					t.Error("Expected JWT secret to be generated")
				}
				if server.config == nil {
					t.Error("Expected config to be loaded")
				}

				expectedURL := tt.config.Server.URL
				if expectedURL == "" {
					expectedURL = "http://localhost:8080"
				}
				if server.serverURL != expectedURL {
					t.Errorf("Expected server URL %s, got %s", expectedURL, server.serverURL)
				}
			}
		})
	}
}

func TestOIDCServer_Handle(t *testing.T) {
	server := createTestOIDCServerForPlugin()

	tests := []struct {
		name           string
		request        shared.HandlerRequest
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "authorize endpoint",
			request: shared.HandlerRequest{
				Method: "GET",
				Path:   "/oidc/authorize",
				Query: url.Values{
					"client_id":     []string{"test-client"},
					"redirect_uri":  []string{"http://localhost:8080/callback"},
					"response_type": []string{"code"},
					"scope":         []string{"openid"},
				},
			},
			expectedStatus: 200,
			expectedBody:   "Sign In", // Should render login form
		},
		{
			name: "token endpoint with invalid method",
			request: shared.HandlerRequest{
				Method: "GET", // Should be POST
				Path:   "/oidc/token",
			},
			expectedStatus: 400,
			expectedBody:   "Only POST method is allowed",
		},
		{
			name: "discovery endpoint",
			request: shared.HandlerRequest{
				Method: "GET",
				Path:   "/.well-known/openid-configuration",
			},
			expectedStatus: 200,
			expectedBody:   "issuer",
		},
		{
			name: "unknown endpoint",
			request: shared.HandlerRequest{
				Method: "GET",
				Path:   "/unknown",
			},
			expectedStatus: 404,
			expectedBody:   "Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := server.Handle(tt.request)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if !strings.Contains(string(resp.Body), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, string(resp.Body))
			}
		})
	}
}

func TestOIDCServer_handleDiscovery(t *testing.T) {
	server := createTestOIDCServerForPlugin()
	server.serverURL = "https://example.com"

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkContent   func(string) bool
	}{
		{
			name:           "GET request returns discovery document",
			method:         "GET",
			expectedStatus: 200,
			checkContent: func(body string) bool {
				return strings.Contains(body, "https://example.com") &&
					strings.Contains(body, "authorization_endpoint") &&
					strings.Contains(body, "token_endpoint") &&
					strings.Contains(body, "userinfo_endpoint") &&
					strings.Contains(body, "issuer")
			},
		},
		{
			name:           "POST request not allowed",
			method:         "POST",
			expectedStatus: 405,
			checkContent: func(body string) bool {
				return strings.Contains(body, "Method Not Allowed")
			},
		},
		{
			name:           "case insensitive method",
			method:         "get",
			expectedStatus: 200,
			checkContent: func(body string) bool {
				return strings.Contains(body, "issuer")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := shared.HandlerRequest{
				Method: tt.method,
				Path:   "/.well-known/openid-configuration",
			}

			resp := server.handleDiscovery(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if !tt.checkContent(string(resp.Body)) {
				t.Errorf("Content check failed for body: %s", string(resp.Body))
			}

			if tt.expectedStatus == 200 {
				contentType := resp.Headers["Content-Type"]
				if contentType != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", contentType)
				}
			}
		})
	}
}

func TestOIDCServer_generateSessionID(t *testing.T) {
	server := createTestOIDCServerForPlugin()

	// Generate multiple session IDs and verify uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := server.generateSessionID()
		if id == "" {
			t.Error("Generated session ID should not be empty")
		}
		if ids[id] {
			t.Errorf("Generated duplicate session ID: %s", id)
		}
		ids[id] = true

		// Basic UUID format check
		if len(id) != 36 {
			t.Errorf("Session ID should be 36 characters, got %d", len(id))
		}
	}
}

func TestOIDCServer_generateAuthCode(t *testing.T) {
	server := createTestOIDCServerForPlugin()

	// Generate multiple auth codes and verify uniqueness
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := server.generateAuthCode()
		if code == "" {
			t.Error("Generated auth code should not be empty")
		}
		if codes[code] {
			t.Errorf("Generated duplicate auth code: %s", code)
		}
		codes[code] = true

		// Should be hex encoded 16 bytes = 32 hex characters
		if len(code) != 32 {
			t.Errorf("Auth code should be 32 characters, got %d", len(code))
		}
	}
}

func TestOIDCServer_generateAccessToken(t *testing.T) {
	server := createTestOIDCServerForPlugin()

	// Generate multiple access tokens and verify uniqueness
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token := server.generateAccessToken()
		if token == "" {
			t.Error("Generated access token should not be empty")
		}
		if tokens[token] {
			t.Errorf("Generated duplicate access token: %s", token)
		}
		tokens[token] = true

		// Basic UUID format check
		if len(token) != 36 {
			t.Errorf("Access token should be 36 characters, got %d", len(token))
		}
	}
}

func TestOIDCServer_cleanupExpired(t *testing.T) {
	server := createTestOIDCServerForPlugin()

	now := time.Now()
	expired := now.Add(-1 * time.Hour)
	notExpired := now.Add(1 * time.Hour)

	// Add test data with mixed expiration times
	server.sessions["expired-session"] = &AuthSession{
		ID:        "expired-session",
		ExpiresAt: expired,
	}
	server.sessions["valid-session"] = &AuthSession{
		ID:        "valid-session",
		ExpiresAt: notExpired,
	}

	server.codes["expired-code"] = &AuthCode{
		Code:      "expired-code",
		ExpiresAt: expired,
	}
	server.codes["valid-code"] = &AuthCode{
		Code:      "valid-code",
		ExpiresAt: notExpired,
	}

	server.tokens["expired-token"] = &AccessToken{
		Token:     "expired-token",
		ExpiresAt: expired,
	}
	server.tokens["valid-token"] = &AccessToken{
		Token:     "valid-token",
		ExpiresAt: notExpired,
	}

	// Run cleanup
	server.cleanupExpired()

	// Check that expired items were removed
	if _, exists := server.sessions["expired-session"]; exists {
		t.Error("Expired session should have been cleaned up")
	}
	if _, exists := server.codes["expired-code"]; exists {
		t.Error("Expired code should have been cleaned up")
	}
	if _, exists := server.tokens["expired-token"]; exists {
		t.Error("Expired token should have been cleaned up")
	}

	// Check that valid items remain
	if _, exists := server.sessions["valid-session"]; !exists {
		t.Error("Valid session should not have been cleaned up")
	}
	if _, exists := server.codes["valid-code"]; !exists {
		t.Error("Valid code should not have been cleaned up")
	}
	if _, exists := server.tokens["valid-token"]; !exists {
		t.Error("Valid token should not have been cleaned up")
	}
}

func TestParseFormData(t *testing.T) {
	tests := []struct {
		name        string
		body        []byte
		expectError bool
		expected    map[string][]string
	}{
		{
			name:        "valid form data",
			body:        []byte("username=alice&password=secret&submit=login"),
			expectError: false,
			expected: map[string][]string{
				"username": {"alice"},
				"password": {"secret"},
				"submit":   {"login"},
			},
		},
		{
			name:        "empty body",
			body:        []byte(""),
			expectError: false,
			expected:    map[string][]string{},
		},
		{
			name:        "URL encoded values",
			body:        []byte("name=John%20Doe&email=john%40example.com"),
			expectError: false,
			expected: map[string][]string{
				"name":  {"John Doe"},
				"email": {"john@example.com"},
			},
		},
		{
			name:        "multiple values for same key",
			body:        []byte("role=admin&role=user&name=test"),
			expectError: false,
			expected: map[string][]string{
				"role": {"admin", "user"},
				"name": {"test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFormData(tt.body)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				for key, expectedValues := range tt.expected {
					actualValues, exists := result[key]
					if !exists {
						t.Errorf("Expected key '%s' to exist in result", key)
						continue
					}
					if len(actualValues) != len(expectedValues) {
						t.Errorf("Expected %d values for key '%s', got %d", len(expectedValues), key, len(actualValues))
						continue
					}
					for i, expectedValue := range expectedValues {
						if actualValues[i] != expectedValue {
							t.Errorf("Expected value '%s' for key '%s'[%d], got '%s'", expectedValue, key, i, actualValues[i])
						}
					}
				}
			}
		})
	}
}

func TestOIDCServer_HandleIntegration(t *testing.T) {
	server := createTestOIDCServerForPlugin()

	t.Run("discovery endpoint provides correct server URL", func(t *testing.T) {
		server.serverURL = "https://custom-server.com"

		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/.well-known/openid-configuration",
		}

		resp := server.Handle(req)

		if resp.StatusCode != 200 {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		body := string(resp.Body)
		if !strings.Contains(body, "https://custom-server.com") {
			t.Error("Discovery document should contain custom server URL")
		}
		if !strings.Contains(body, "https://custom-server.com/oidc/authorize") {
			t.Error("Discovery document should contain correct authorization endpoint")
		}
		if !strings.Contains(body, "https://custom-server.com/oidc/token") {
			t.Error("Discovery document should contain correct token endpoint")
		}
	})
}
