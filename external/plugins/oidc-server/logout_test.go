package main

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
)

func createTestOIDCServerForLogout() *OIDCServer {
	config := &OIDCConfig{
		Users: []User{
			{
				Username: "alice",
				Password: "password",
				Claims: map[string]string{
					"sub":   "alice",
					"email": "alice@example.com",
				},
			},
			{
				Username: "bob",
				Password: "password",
				Claims: map[string]string{
					"sub":   "bob",
					"email": "bob@example.com",
				},
			},
		},
		Clients: []Client{
			{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				RedirectURIs: []string{"http://localhost:8080/callback"},
				PostLogoutRedirectURIs: []string{
					"http://localhost:8080/logged-out",
					"http://localhost:8080/goodbye",
				},
			},
			{
				ClientID:     "other-client",
				ClientSecret: "other-secret",
				RedirectURIs: []string{"http://localhost:9090/callback"},
				PostLogoutRedirectURIs: []string{
					"http://localhost:9090/logged-out",
				},
			},
		},
		JWTConfig: &JWTConfig{
			Algorithm: "HS256",
			Secret:    "test-secret-key-for-deterministic-jwt-signing-in-logout-tests",
		},
	}
	config.PathPrefix = "/oidc"

	server := &OIDCServer{
		logger:     hclog.NewNullLogger(),
		config:     config,
		serverURL:  "http://localhost:8080",
		pathPrefix: config.PathPrefix,
		sessions:   make(map[string]*AuthSession),
		codes:      make(map[string]*AuthCode),
		tokens:     make(map[string]*AccessToken),
	}

	server.setupJWTKeys()
	server.CacheDiscoveryDocument()

	return server
}

func generateTestIDToken(server *OIDCServer, sub, aud string, expired bool) string {
	now := time.Now()
	exp := now.Add(1 * time.Hour)
	if expired {
		now = now.Add(-2 * time.Hour)
		exp = now.Add(1 * time.Hour) // still in the past
	}

	claims := jwt.MapClaims{
		"iss": server.serverURL,
		"sub": sub,
		"aud": aud,
		"iat": now.Unix(),
		"exp": exp.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString(server.jwtSecret)
	return signed
}

func TestOIDCServer_handleLogout_MethodDispatch(t *testing.T) {
	server := createTestOIDCServerForLogout()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET returns 200 confirmation",
			method:         "GET",
			expectedStatus: 200,
		},
		{
			name:           "POST returns 200 confirmation",
			method:         "POST",
			expectedStatus: 200,
		},
		{
			name:           "PUT returns 405",
			method:         "PUT",
			expectedStatus: 405,
		},
		{
			name:           "DELETE returns 405",
			method:         "DELETE",
			expectedStatus: 405,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := shared.HandlerRequest{
				Method: tt.method,
				Path:   "/oidc/logout",
				Query:  url.Values{},
			}

			resp := server.handleLogout(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestOIDCServer_handleLogout_ParameterValidation(t *testing.T) {
	server := createTestOIDCServerForLogout()

	tests := []struct {
		name           string
		query          url.Values
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "no params returns confirmation page",
			query:          url.Values{},
			expectedStatus: 200,
			expectedBody:   "Signed Out",
		},
		{
			name: "post_logout_redirect_uri without client identification returns 400",
			query: url.Values{
				"post_logout_redirect_uri": []string{"http://localhost:8080/logged-out"},
			},
			expectedStatus: 400,
			expectedBody:   "client_id or id_token_hint is required",
		},
		{
			name: "unregistered post_logout_redirect_uri returns 400",
			query: url.Values{
				"post_logout_redirect_uri": []string{"http://evil.com/steal"},
				"client_id":                []string{"test-client"},
			},
			expectedStatus: 400,
			expectedBody:   "Invalid post_logout_redirect_uri",
		},
		{
			name: "unknown client returns 400",
			query: url.Values{
				"post_logout_redirect_uri": []string{"http://localhost:8080/logged-out"},
				"client_id":                []string{"nonexistent-client"},
			},
			expectedStatus: 400,
			expectedBody:   "Unknown client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := shared.HandlerRequest{
				Method: "GET",
				Path:   "/oidc/logout",
				Query:  tt.query,
			}

			resp := server.handleLogout(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if !strings.Contains(string(resp.Body), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, string(resp.Body))
			}
		})
	}
}

func TestOIDCServer_handleLogout_IDTokenHint(t *testing.T) {
	server := createTestOIDCServerForLogout()

	t.Run("valid token extracts client and user", func(t *testing.T) {
		idToken := generateTestIDToken(server, "alice", "test-client", false)

		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/logout",
			Query: url.Values{
				"id_token_hint":            []string{idToken},
				"post_logout_redirect_uri": []string{"http://localhost:8080/logged-out"},
			},
		}

		resp := server.handleLogout(req)

		if resp.StatusCode != 302 {
			t.Errorf("Expected 302 redirect, got %d: %s", resp.StatusCode, string(resp.Body))
		}
	})

	t.Run("expired token still accepted", func(t *testing.T) {
		idToken := generateTestIDToken(server, "alice", "test-client", true)

		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/logout",
			Query: url.Values{
				"id_token_hint":            []string{idToken},
				"post_logout_redirect_uri": []string{"http://localhost:8080/logged-out"},
			},
		}

		resp := server.handleLogout(req)

		if resp.StatusCode != 302 {
			t.Errorf("Expected 302 redirect for expired token, got %d: %s", resp.StatusCode, string(resp.Body))
		}
	})

	t.Run("invalid signature shows confirmation instead of error", func(t *testing.T) {
		// Token signed with wrong key
		claims := jwt.MapClaims{
			"iss": server.serverURL,
			"sub": "alice",
			"aud": "test-client",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		badToken, _ := token.SignedString([]byte("wrong-secret-key-that-does-not-match-the-configured-one"))

		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/logout",
			Query: url.Values{
				"id_token_hint": []string{badToken},
			},
		}

		resp := server.handleLogout(req)

		// Without post_logout_redirect_uri, should show confirmation page
		if resp.StatusCode != 200 {
			t.Errorf("Expected 200 confirmation, got %d", resp.StatusCode)
		}
		if !strings.Contains(string(resp.Body), "Signed Out") {
			t.Error("Expected confirmation page with 'Signed Out'")
		}
	})

	t.Run("explicit client_id takes precedence over token aud", func(t *testing.T) {
		idToken := generateTestIDToken(server, "alice", "test-client", false)

		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/logout",
			Query: url.Values{
				"id_token_hint":            []string{idToken},
				"client_id":                []string{"other-client"},
				"post_logout_redirect_uri": []string{"http://localhost:9090/logged-out"},
			},
		}

		resp := server.handleLogout(req)

		if resp.StatusCode != 302 {
			t.Errorf("Expected 302 redirect, got %d: %s", resp.StatusCode, string(resp.Body))
		}
		location := resp.Headers["Location"]
		if !strings.HasPrefix(location, "http://localhost:9090/logged-out") {
			t.Errorf("Expected redirect to other-client URI, got %s", location)
		}
	})
}

func TestOIDCServer_handleLogout_RedirectFlow(t *testing.T) {
	server := createTestOIDCServerForLogout()

	t.Run("valid redirect with client_id", func(t *testing.T) {
		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/logout",
			Query: url.Values{
				"client_id":                []string{"test-client"},
				"post_logout_redirect_uri": []string{"http://localhost:8080/logged-out"},
			},
		}

		resp := server.handleLogout(req)

		if resp.StatusCode != 302 {
			t.Errorf("Expected 302, got %d", resp.StatusCode)
		}
		location := resp.Headers["Location"]
		if location != "http://localhost:8080/logged-out" {
			t.Errorf("Expected redirect to logged-out URI, got %s", location)
		}
		if resp.Headers["Cache-Control"] != "no-store" {
			t.Error("Expected Cache-Control: no-store header")
		}
	})

	t.Run("state passed through in redirect", func(t *testing.T) {
		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/logout",
			Query: url.Values{
				"client_id":                []string{"test-client"},
				"post_logout_redirect_uri": []string{"http://localhost:8080/logged-out"},
				"state":                    []string{"my-logout-state"},
			},
		}

		resp := server.handleLogout(req)

		if resp.StatusCode != 302 {
			t.Errorf("Expected 302, got %d", resp.StatusCode)
		}

		location := resp.Headers["Location"]
		u, err := url.Parse(location)
		if err != nil {
			t.Fatalf("Failed to parse redirect location: %v", err)
		}
		if u.Query().Get("state") != "my-logout-state" {
			t.Errorf("Expected state 'my-logout-state' in redirect, got '%s'", u.Query().Get("state"))
		}
	})

	t.Run("URI extracted from id_token_hint aud claim works", func(t *testing.T) {
		idToken := generateTestIDToken(server, "alice", "test-client", false)

		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/logout",
			Query: url.Values{
				"id_token_hint":            []string{idToken},
				"post_logout_redirect_uri": []string{"http://localhost:8080/logged-out"},
			},
		}

		resp := server.handleLogout(req)

		if resp.StatusCode != 302 {
			t.Errorf("Expected 302, got %d: %s", resp.StatusCode, string(resp.Body))
		}
	})
}

func TestOIDCServer_handleLogout_StateCleanup(t *testing.T) {
	t.Run("tokens for identified user cleared", func(t *testing.T) {
		server := createTestOIDCServerForLogout()

		// Add tokens for alice
		server.tokens["alice-token-1"] = &AccessToken{
			Token:     "alice-token-1",
			UserID:    "alice",
			ClientID:  "test-client",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		server.tokens["alice-token-2"] = &AccessToken{
			Token:     "alice-token-2",
			UserID:    "alice",
			ClientID:  "test-client",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		// Add tokens for bob (should not be cleared)
		server.tokens["bob-token"] = &AccessToken{
			Token:     "bob-token",
			UserID:    "bob",
			ClientID:  "test-client",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		// Add codes for alice
		server.codes["alice-code"] = &AuthCode{
			Code:      "alice-code",
			UserID:    "alice",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}

		idToken := generateTestIDToken(server, "alice", "test-client", false)

		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/logout",
			Query: url.Values{
				"id_token_hint": []string{idToken},
			},
		}

		server.handleLogout(req)

		// Alice's tokens and codes should be cleared
		server.mutex.RLock()
		defer server.mutex.RUnlock()

		if _, exists := server.tokens["alice-token-1"]; exists {
			t.Error("Expected alice-token-1 to be cleared")
		}
		if _, exists := server.tokens["alice-token-2"]; exists {
			t.Error("Expected alice-token-2 to be cleared")
		}
		if _, exists := server.codes["alice-code"]; exists {
			t.Error("Expected alice-code to be cleared")
		}

		// Bob's tokens should remain
		if _, exists := server.tokens["bob-token"]; !exists {
			t.Error("Expected bob-token to be preserved")
		}
	})

	t.Run("empty userID clears nothing", func(t *testing.T) {
		server := createTestOIDCServerForLogout()

		server.tokens["some-token"] = &AccessToken{
			Token:     "some-token",
			UserID:    "alice",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		// Logout without id_token_hint means no userID
		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/logout",
			Query:  url.Values{},
		}

		server.handleLogout(req)

		server.mutex.RLock()
		defer server.mutex.RUnlock()

		if _, exists := server.tokens["some-token"]; !exists {
			t.Error("Expected token to be preserved when no user identified")
		}
	})
}

func TestOIDCServer_parseIDTokenHint(t *testing.T) {
	server := createTestOIDCServerForLogout()

	t.Run("valid HS256 token", func(t *testing.T) {
		idToken := generateTestIDToken(server, "alice", "test-client", false)

		clientID, sub, err := server.parseIDTokenHint(idToken)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if clientID != "test-client" {
			t.Errorf("Expected clientID 'test-client', got '%s'", clientID)
		}
		if sub != "alice" {
			t.Errorf("Expected sub 'alice', got '%s'", sub)
		}
	})

	t.Run("expired token still accepted", func(t *testing.T) {
		idToken := generateTestIDToken(server, "bob", "test-client", true)

		clientID, sub, err := server.parseIDTokenHint(idToken)
		if err != nil {
			t.Fatalf("Unexpected error for expired token: %v", err)
		}
		if clientID != "test-client" {
			t.Errorf("Expected clientID 'test-client', got '%s'", clientID)
		}
		if sub != "bob" {
			t.Errorf("Expected sub 'bob', got '%s'", sub)
		}
	})

	t.Run("malformed token returns error", func(t *testing.T) {
		_, _, err := server.parseIDTokenHint("not.a.valid.token")
		if err == nil {
			t.Error("Expected error for malformed token")
		}
	})

	t.Run("wrong key returns error", func(t *testing.T) {
		claims := jwt.MapClaims{
			"iss": server.serverURL,
			"sub": "alice",
			"aud": "test-client",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		badToken, _ := token.SignedString([]byte("wrong-secret-key-that-does-not-match-the-configured-one"))

		_, _, err := server.parseIDTokenHint(badToken)
		if err == nil {
			t.Error("Expected error for token signed with wrong key")
		}
	})

	t.Run("wrong issuer returns error", func(t *testing.T) {
		claims := jwt.MapClaims{
			"iss": "http://evil-server.com",
			"sub": "alice",
			"aud": "test-client",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		badToken, _ := token.SignedString(server.jwtSecret)

		_, _, err := server.parseIDTokenHint(badToken)
		if err == nil {
			t.Error("Expected error for token with wrong issuer")
		}
		if !strings.Contains(err.Error(), "issuer mismatch") {
			t.Errorf("Expected issuer mismatch error, got: %v", err)
		}
	})
}

func TestOIDCServer_parseIDTokenHint_RS256(t *testing.T) {
	// Create an RS256-configured server
	config := &OIDCConfig{
		Users: []User{
			{
				Username: "alice",
				Password: "password",
				Claims:   map[string]string{"sub": "alice"},
			},
		},
		Clients: []Client{
			{
				ClientID:               "test-client",
				ClientSecret:           "test-secret",
				RedirectURIs:           []string{"http://localhost:8080/callback"},
				PostLogoutRedirectURIs: []string{"http://localhost:8080/logged-out"},
			},
		},
		JWTConfig: &JWTConfig{
			Algorithm: "RS256",
		},
	}
	config.PathPrefix = "/oidc"

	server := &OIDCServer{
		logger:     hclog.NewNullLogger(),
		config:     config,
		serverURL:  "http://localhost:8080",
		pathPrefix: config.PathPrefix,
		sessions:   make(map[string]*AuthSession),
		codes:      make(map[string]*AuthCode),
		tokens:     make(map[string]*AccessToken),
	}
	server.setupJWTKeys()
	server.CacheDiscoveryDocument()

	t.Run("valid RS256 token", func(t *testing.T) {
		// Generate an RS256 ID token
		user := config.Users[0]
		scopes := []string{"openid"}
		idTokenStr, err := server.generateIDToken(&user, "test-client", "", scopes, time.Now(), 3600)
		if err != nil {
			t.Fatalf("Failed to generate ID token: %v", err)
		}

		clientID, sub, err := server.parseIDTokenHint(idTokenStr)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if clientID != "test-client" {
			t.Errorf("Expected clientID 'test-client', got '%s'", clientID)
		}
		if sub != "alice" {
			t.Errorf("Expected sub 'alice', got '%s'", sub)
		}
	})
}
