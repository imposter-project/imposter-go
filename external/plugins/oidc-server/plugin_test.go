package main

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
)

// handleRequest simulates the NormaliseRequest → TransformResponse flow
// that the core ExternalPluginHandler orchestrates.
func (o *OIDCServer) handleRequest(args shared.HandlerRequest) shared.TransformResponseResult {
	normResp, _ := o.NormaliseRequest(args)
	if normResp.Skip {
		return shared.TransformResponseResult{StatusCode: 0}
	}
	result, _ := o.TransformResponse(shared.TransformRequest{
		Method:   args.Method,
		Path:     args.Path,
		Query:    args.Query,
		Headers:  args.Headers,
		Body:     args.Body,
		Handled:  false,
		Metadata: normResp.Metadata,
	})
	return result
}

func createTestOIDCServerForPlugin() *OIDCServer {
	config := getDefaultConfig()
	server := &OIDCServer{
		logger: hclog.New(&hclog.LoggerOptions{
			Level:  hclog.Off, // Disable logging during tests
			Output: nil,
		}),
		config:     config,
		serverURL:  "http://localhost:8080",
		pathPrefix: config.PathPrefix,
		sessions:   make(map[string]*AuthSession),
		codes:      make(map[string]*AuthCode),
		tokens:     make(map[string]*AccessToken),
	}

	// Setup JWT keys based on the default config (RS256)
	server.setupJWTKeys()
	server.CacheDiscoveryDocument()

	return server
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

			_, err := server.Configure(tt.config)

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

				// Check JWT credentials based on algorithm
				if server.config.JWTConfig.Algorithm == "HS256" {
					if server.jwtSecret == nil {
						t.Error("Expected JWT secret to be generated for HS256")
					}
				} else if server.config.JWTConfig.Algorithm == "RS256" {
					if server.privateKey == nil {
						t.Error("Expected private key to be loaded for RS256")
					}
					if server.publicKey == nil {
						t.Error("Expected public key to be loaded for RS256")
					}
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
	server.CacheDiscoveryDocument() // Cache discovery document for Handle tests

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
				Path:   "/oidc/.well-known/openid-configuration",
			},
			expectedStatus: 200,
			expectedBody:   "issuer",
		},
		{
			name: "logout endpoint",
			request: shared.HandlerRequest{
				Method: "GET",
				Path:   "/oidc/logout",
				Query:  url.Values{},
			},
			expectedStatus: 200,
			expectedBody:   "Signed Out",
		},
		{
			name: "unknown endpoint",
			request: shared.HandlerRequest{
				Method: "GET",
				Path:   "/unknown",
			},
			expectedStatus: 0,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := server.handleRequest(tt.request)

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
	server.CacheDiscoveryDocument() // Cache after setting server URL

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
					strings.Contains(body, "end_session_endpoint") &&
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
				Path:   "/oidc/.well-known/openid-configuration",
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
		server.CacheDiscoveryDocument() // Cache after setting custom server URL

		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/.well-known/openid-configuration",
		}

		resp := server.handleRequest(req)

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
		if !strings.Contains(body, "https://custom-server.com/oidc/logout") {
			t.Error("Discovery document should contain correct end_session_endpoint")
		}
	})
}

// The issuer field must exactly match the URL a client would fetch the
// discovery document from:
// https://datatracker.ietf.org/doc/html/rfc8414#section-3.3
// This was previously hardcoded to serverURL alone, omitting pathPrefix,
// while every other endpoint in the same document correctly included it.
func TestOIDCServer_CacheDiscoveryDocument_IssuerIncludesPathPrefix(t *testing.T) {
	t.Run("default path prefix", func(t *testing.T) {
		server := createTestOIDCServerForPlugin()
		server.serverURL = "http://localhost:8080"
		if err := server.CacheDiscoveryDocument(); err != nil {
			t.Fatalf("Failed to cache discovery document: %v", err)
		}

		var discovery map[string]interface{}
		if err := json.Unmarshal(server.cachedDiscovery, &discovery); err != nil {
			t.Fatalf("Failed to parse discovery document: %v", err)
		}

		expectedIssuer := "http://localhost:8080/oidc"
		if discovery["issuer"] != expectedIssuer {
			t.Errorf("Expected issuer %q, got %q", expectedIssuer, discovery["issuer"])
		}
	})

	t.Run("custom path prefix", func(t *testing.T) {
		server := createTestOIDCServerForPlugin()
		server.serverURL = "https://custom-server.com"
		server.pathPrefix = "/custom-oidc"
		if err := server.CacheDiscoveryDocument(); err != nil {
			t.Fatalf("Failed to cache discovery document: %v", err)
		}

		var discovery map[string]interface{}
		if err := json.Unmarshal(server.cachedDiscovery, &discovery); err != nil {
			t.Fatalf("Failed to parse discovery document: %v", err)
		}

		expectedIssuer := "https://custom-server.com/custom-oidc"
		if discovery["issuer"] != expectedIssuer {
			t.Errorf("Expected issuer %q, got %q", expectedIssuer, discovery["issuer"])
		}
	})
}

// TestOIDCServer_IssuerConsistency_AcrossComponents locks the three issuer
// surfaces together so they can never silently drift apart again:
//  1. the discovery document's "issuer" (plugin.go)
//  2. the "iss" claim minted into ID tokens (token.go)
//  3. the issuer the RP-initiated logout flow validates against (logout.go)
//
// The original bug (see TestOIDCServer_CacheDiscoveryDocument_IssuerIncludesPathPrefix)
// slipped past every existing test because each surface independently used the
// same wrong value, so they stayed internally consistent with one another while
// collectively disagreeing with the URL clients actually use. Asserting each
// value against a hardcoded literal in isolation would not have caught it. This
// test instead asserts the relationships between the surfaces, under a
// non-default path prefix, so changing any single one of them breaks it.
func TestOIDCServer_IssuerConsistency_AcrossComponents(t *testing.T) {
	config := &OIDCConfig{
		PathPrefix: "/custom-oidc",
		Users: []User{
			{Username: "alice", Password: "password", Claims: map[string]string{"sub": "alice"}},
		},
		Clients: []Client{
			{
				ClientID:               "test-client",
				ClientSecret:           "test-secret",
				RedirectURIs:           []string{"https://issuer.example.com/callback"},
				PostLogoutRedirectURIs: []string{"https://issuer.example.com/logged-out"},
			},
		},
		JWTConfig: &JWTConfig{Algorithm: "RS256"},
	}

	server := &OIDCServer{
		logger:     hclog.NewNullLogger(),
		config:     config,
		serverURL:  "https://issuer.example.com",
		pathPrefix: config.PathPrefix,
		sessions:   make(map[string]*AuthSession),
		codes:      make(map[string]*AuthCode),
		tokens:     make(map[string]*AccessToken),
	}
	if err := server.setupJWTKeys(); err != nil {
		t.Fatalf("Failed to setup JWT keys: %v", err)
	}
	if err := server.CacheDiscoveryDocument(); err != nil {
		t.Fatalf("Failed to cache discovery document: %v", err)
	}

	// 1. Issuer advertised by the discovery document.
	var discovery map[string]interface{}
	if err := json.Unmarshal(server.cachedDiscovery, &discovery); err != nil {
		t.Fatalf("Failed to parse discovery document: %v", err)
	}
	discoveryIssuer, _ := discovery["issuer"].(string)

	// It must include the configured path prefix — the location the discovery
	// document is actually served from.
	wantIssuer := "https://issuer.example.com/custom-oidc"
	if discoveryIssuer != wantIssuer {
		t.Errorf("discovery issuer = %q, want %q", discoveryIssuer, wantIssuer)
	}

	// 2. The iss claim minted into an ID token must equal the discovery issuer.
	user := config.Users[0]
	idToken, err := server.generateIDToken(&user, "test-client", "nonce-123", []string{"openid"}, time.Now(), 3600)
	if err != nil {
		t.Fatalf("Failed to generate ID token: %v", err)
	}
	parsed, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		return server.publicKey, nil
	})
	if err != nil {
		t.Fatalf("Failed to parse ID token: %v", err)
	}
	tokenIss, _ := parsed.Claims.(jwt.MapClaims)["iss"].(string)
	if tokenIss != discoveryIssuer {
		t.Errorf("ID token iss = %q, but discovery issuer = %q; they must be identical", tokenIss, discoveryIssuer)
	}

	// 3. The RP-initiated logout flow must accept a token this same server minted.
	//    A stale issuer check here would reject genuine tokens with an "issuer
	//    mismatch" error.
	clientID, sub, err := server.parseIDTokenHint(idToken)
	if err != nil {
		t.Fatalf("logout rejected a token minted by this server: %v", err)
	}
	if clientID != "test-client" {
		t.Errorf("parseIDTokenHint clientID = %q, want %q", clientID, "test-client")
	}
	if sub != "alice" {
		t.Errorf("parseIDTokenHint sub = %q, want %q", sub, "alice")
	}
}
