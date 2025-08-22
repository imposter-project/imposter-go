package main

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
)

// TestOIDCServer_EndToEndFlow tests the complete OIDC authorization code flow
// from initial authorization request through token exchange to userinfo retrieval
func TestOIDCServer_EndToEndFlow(t *testing.T) {
	// Setup: Create server with static secret for deterministic testing
	config := &OIDCConfig{
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
		},
		Clients: []Client{
			{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				RedirectURIs: []string{"http://localhost:8080/callback"},
			},
		},
		JWTConfig: &JWTConfig{
			Algorithm: "HS256",
			Secret:    "test-secret-key-for-deterministic-jwt-signing-in-end-to-end-test",
		},
	}

	server := &OIDCServer{
		logger:    hclog.NewNullLogger(),
		config:    config,
		serverURL: "http://localhost:8080",
		sessions:  make(map[string]*AuthSession),
		codes:     make(map[string]*AuthCode),
		tokens:    make(map[string]*AccessToken),
	}

	// Initialize JWT setup
	err := server.setupJWTKeys()
	if err != nil {
		t.Fatalf("Failed to setup JWT keys: %v", err)
	}

	// Cache discovery document for endpoint testing
	err = server.CacheDiscoveryDocument()
	if err != nil {
		t.Fatalf("Failed to cache discovery document: %v", err)
	}

	t.Run("complete OIDC authorization code flow", func(t *testing.T) {
		// Step 1: Discovery - Client discovers OIDC endpoints
		t.Log("Step 1: OIDC Discovery")
		discoveryReq := shared.HandlerRequest{
			Method: "GET",
			Path:   "/.well-known/openid-configuration",
		}

		discoveryResp := server.handleDiscovery(discoveryReq)
		if discoveryResp.StatusCode != 200 {
			t.Fatalf("Discovery failed with status %d", discoveryResp.StatusCode)
		}

		var discoveryDoc map[string]interface{}
		if err := json.Unmarshal(discoveryResp.Body, &discoveryDoc); err != nil {
			t.Fatalf("Failed to parse discovery document: %v", err)
		}

		// Verify discovery document contains required endpoints
		requiredEndpoints := []string{"authorization_endpoint", "token_endpoint", "userinfo_endpoint"}
		for _, endpoint := range requiredEndpoints {
			if _, exists := discoveryDoc[endpoint]; !exists {
				t.Errorf("Discovery document missing %s", endpoint)
			}
		}
		t.Log("✓ Discovery document validated")

		// Step 2: Authorization Request - Client redirects user to authorization endpoint
		t.Log("Step 2: Authorization Request")
		authReq := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/authorize",
			Query: url.Values{
				"client_id":     []string{"test-client"},
				"redirect_uri":  []string{"http://localhost:8080/callback"},
				"response_type": []string{"code"},
				"scope":         []string{"openid profile email"},
				"state":         []string{"e2e-test-state-12345"},
				"nonce":         []string{"e2e-test-nonce-67890"},
			},
		}

		authResp := server.handleAuthorizeGet(authReq)
		if authResp.StatusCode != 200 {
			t.Fatalf("Authorization request failed with status %d", authResp.StatusCode)
		}

		// Extract session ID from login form
		loginForm := string(authResp.Body)
		if !strings.Contains(loginForm, "name=\"username\"") || !strings.Contains(loginForm, "name=\"password\"") {
			t.Fatal("Login form missing required fields")
		}

		sessionID := extractSessionID(t, loginForm)
		t.Logf("✓ Authorization request created session: %s", sessionID)

		// Step 3: User Authentication - User submits credentials
		t.Log("Step 3: User Authentication")
		loginReq := shared.HandlerRequest{
			Method: "POST",
			Path:   "/oidc/authorize",
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			Body: []byte("session_id=" + sessionID + "&username=alice&password=password"),
		}

		loginResp := server.handleAuthorizePost(loginReq)
		if loginResp.StatusCode != 302 {
			t.Fatalf("User authentication failed with status %d", loginResp.StatusCode)
		}

		// Extract authorization code from redirect
		location := loginResp.Headers["Location"]
		if !strings.Contains(location, "http://localhost:8080/callback") {
			t.Fatalf("Invalid redirect URI: %s", location)
		}

		authCode := extractAuthCodeFromLocation(t, location)
		state := extractStateFromLocation(t, location)

		if state != "e2e-test-state-12345" {
			t.Errorf("State mismatch: expected 'e2e-test-state-12345', got '%s'", state)
		}
		t.Logf("✓ User authenticated, authorization code: %s", authCode)

		// Step 4: Token Exchange - Client exchanges authorization code for tokens
		t.Log("Step 4: Token Exchange")
		tokenReq := shared.HandlerRequest{
			Method: "POST",
			Path:   "/oidc/token",
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			Body: []byte("grant_type=authorization_code&code=" + authCode + "&client_id=test-client&client_secret=test-secret&redirect_uri=http://localhost:8080/callback"),
		}

		tokenResp := server.handleToken(tokenReq)
		if tokenResp.StatusCode != 200 {
			t.Fatalf("Token exchange failed with status %d: %s", tokenResp.StatusCode, string(tokenResp.Body))
		}

		var tokenResponse TokenResponse
		if err := json.Unmarshal(tokenResp.Body, &tokenResponse); err != nil {
			t.Fatalf("Failed to parse token response: %v", err)
		}

		// Validate token response
		if tokenResponse.AccessToken == "" {
			t.Error("Missing access token")
		}
		if tokenResponse.IDToken == "" {
			t.Error("Missing ID token")
		}
		if tokenResponse.TokenType != "Bearer" {
			t.Errorf("Expected token_type 'Bearer', got '%s'", tokenResponse.TokenType)
		}
		if tokenResponse.ExpiresIn <= 0 {
			t.Errorf("Invalid expires_in: %d", tokenResponse.ExpiresIn)
		}

		t.Logf("✓ Tokens received - Access: %s..., ID: %s...",
			tokenResponse.AccessToken[:10], tokenResponse.IDToken[:10])

		// Step 5: ID Token Validation - Verify ID token claims
		t.Log("Step 5: ID Token Validation")
		idToken, err := jwt.Parse(tokenResponse.IDToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				t.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(config.JWTConfig.Secret), nil
		})

		if err != nil {
			t.Fatalf("Failed to parse ID token: %v", err)
		}

		if !idToken.Valid {
			t.Fatal("ID token is invalid")
		}

		claims, ok := idToken.Claims.(jwt.MapClaims)
		if !ok {
			t.Fatal("Failed to extract claims from ID token")
		}

		// Validate required claims
		requiredClaims := map[string]string{
			"sub":   "alice",
			"aud":   "test-client",
			"iss":   "http://localhost:8080",
			"nonce": "e2e-test-nonce-67890",
		}

		for claimName, expectedValue := range requiredClaims {
			if claimValue, exists := claims[claimName]; !exists {
				t.Errorf("Missing claim: %s", claimName)
			} else if claimValue != expectedValue {
				t.Errorf("Claim %s: expected '%s', got '%v'", claimName, expectedValue, claimValue)
			}
		}

		// Validate profile claims from scope
		profileClaims := []string{"name", "given_name", "family_name"}
		for _, claimName := range profileClaims {
			if _, exists := claims[claimName]; !exists {
				t.Errorf("Missing profile claim: %s", claimName)
			}
		}

		t.Log("✓ ID token validated with all expected claims")

		// Step 6: UserInfo Request - Client retrieves user information
		t.Log("Step 6: UserInfo Request")
		userinfoReq := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/userinfo",
			Headers: map[string]string{
				"Authorization": "Bearer " + tokenResponse.AccessToken,
			},
		}

		userinfoResp := server.handleUserInfo(userinfoReq)
		if userinfoResp.StatusCode != 200 {
			t.Fatalf("UserInfo request failed with status %d: %s", userinfoResp.StatusCode, string(userinfoResp.Body))
		}

		var userInfo map[string]interface{}
		if err := json.Unmarshal(userinfoResp.Body, &userInfo); err != nil {
			t.Fatalf("Failed to parse UserInfo response: %v", err)
		}

		// Validate UserInfo response
		expectedUserInfo := map[string]string{
			"sub":         "alice",
			"name":        "Alice Smith",
			"given_name":  "Alice",
			"family_name": "Smith",
			"email":       "alice@example.com",
		}

		for key, expectedValue := range expectedUserInfo {
			if actualValue, exists := userInfo[key]; !exists {
				t.Errorf("Missing UserInfo claim: %s", key)
			} else if actualValue != expectedValue {
				t.Errorf("UserInfo claim %s: expected '%s', got '%v'", key, expectedValue, actualValue)
			}
		}

		t.Log("✓ UserInfo retrieved and validated")

		// Step 7: Cleanup Validation - Ensure proper cleanup
		t.Log("Step 7: Cleanup Validation")
		server.mutex.RLock()
		sessionCount := len(server.sessions)
		codeCount := len(server.codes)   // Should be 0 after token exchange
		tokenCount := len(server.tokens) // Should be 1 (our access token)
		server.mutex.RUnlock()

		if sessionCount != 0 {
			t.Errorf("Expected 0 sessions after flow completion, got %d", sessionCount)
		}
		if codeCount != 0 {
			t.Errorf("Expected 0 authorization codes after token exchange, got %d", codeCount)
		}
		if tokenCount != 1 {
			t.Errorf("Expected 1 access token, got %d", tokenCount)
		}

		t.Log("✓ Cleanup validation passed")
		t.Log("🎉 End-to-end OIDC flow completed successfully!")
	})
}

// TestOIDCServer_EndToEndFlowWithPKCE tests the complete flow with PKCE enabled
func TestOIDCServer_EndToEndFlowWithPKCE(t *testing.T) {
	// Setup server with HS256 and static secret
	config := &OIDCConfig{
		Users: []User{
			{
				Username: "bob",
				Password: "password",
				Claims: map[string]string{
					"sub":   "bob",
					"name":  "Bob Jones",
					"email": "bob@example.com",
				},
			},
		},
		Clients: []Client{
			{
				ClientID:     "mobile-app", // No client secret for PKCE flow
				RedirectURIs: []string{"com.example.app://oauth/callback"},
			},
		},
		JWTConfig: &JWTConfig{
			Algorithm: "HS256",
			Secret:    "pkce-test-secret-key-for-deterministic-jwt-signing-in-end-to-end-test",
		},
	}

	server := &OIDCServer{
		logger:    hclog.NewNullLogger(),
		config:    config,
		serverURL: "http://localhost:8080",
		sessions:  make(map[string]*AuthSession),
		codes:     make(map[string]*AuthCode),
		tokens:    make(map[string]*AccessToken),
	}

	err := server.setupJWTKeys()
	if err != nil {
		t.Fatalf("Failed to setup JWT keys: %v", err)
	}

	err = server.CacheDiscoveryDocument()
	if err != nil {
		t.Fatalf("Failed to cache discovery document: %v", err)
	}

	t.Run("complete OIDC flow with PKCE", func(t *testing.T) {
		// Generate PKCE parameters
		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		codeChallenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" // S256 of above

		t.Log("Step 1: PKCE Authorization Request")
		authReq := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/authorize",
			Query: url.Values{
				"client_id":             []string{"mobile-app"},
				"redirect_uri":          []string{"com.example.app://oauth/callback"},
				"response_type":         []string{"code"},
				"scope":                 []string{"openid profile"},
				"state":                 []string{"pkce-test-state"},
				"code_challenge":        []string{codeChallenge},
				"code_challenge_method": []string{"S256"},
			},
		}

		authResp := server.handleAuthorizeGet(authReq)
		if authResp.StatusCode != 200 {
			t.Fatalf("PKCE authorization request failed: %d", authResp.StatusCode)
		}

		sessionID := extractSessionID(t, string(authResp.Body))
		t.Log("✓ PKCE authorization request created session")

		// User authentication
		t.Log("Step 2: User Authentication")
		loginReq := shared.HandlerRequest{
			Method: "POST",
			Path:   "/oidc/authorize",
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			Body: []byte("session_id=" + sessionID + "&username=bob&password=password"),
		}

		loginResp := server.handleAuthorizePost(loginReq)
		if loginResp.StatusCode != 302 {
			t.Fatalf("PKCE authentication failed: %d", loginResp.StatusCode)
		}

		authCode := extractAuthCodeFromLocation(t, loginResp.Headers["Location"])
		t.Log("✓ PKCE authentication successful")

		// Token exchange with PKCE
		t.Log("Step 3: PKCE Token Exchange")
		tokenReq := shared.HandlerRequest{
			Method: "POST",
			Path:   "/oidc/token",
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			Body: []byte("grant_type=authorization_code&code=" + authCode + "&client_id=mobile-app&redirect_uri=com.example.app://oauth/callback&code_verifier=" + codeVerifier),
		}

		tokenResp := server.handleToken(tokenReq)
		if tokenResp.StatusCode != 200 {
			t.Fatalf("PKCE token exchange failed: %d: %s", tokenResp.StatusCode, string(tokenResp.Body))
		}

		var tokenResponse TokenResponse
		if err := json.Unmarshal(tokenResp.Body, &tokenResponse); err != nil {
			t.Fatalf("Failed to parse PKCE token response: %v", err)
		}

		if tokenResponse.AccessToken == "" || tokenResponse.IDToken == "" {
			t.Error("Missing tokens in PKCE flow")
		}

		t.Log("✓ PKCE token exchange successful")
		t.Log("🎉 End-to-end OIDC flow with PKCE completed successfully!")
	})
}

// Helper functions
func extractSessionID(t *testing.T, loginForm string) string {
	sessionMarker := "name=\"session_id\" value=\""
	sessionStart := strings.Index(loginForm, sessionMarker)
	if sessionStart == -1 {
		t.Fatal("Could not find session_id field in login form")
	}
	sessionStart += len(sessionMarker)
	sessionEnd := strings.Index(loginForm[sessionStart:], "\"")
	if sessionEnd == -1 {
		t.Fatal("Could not find end of session_id value")
	}
	sessionEnd += sessionStart
	return loginForm[sessionStart:sessionEnd]
}

func extractAuthCodeFromLocation(t *testing.T, location string) string {
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("Failed to parse redirect location: %v", err)
	}

	code := u.Query().Get("code")
	if code == "" {
		t.Fatal("No authorization code found in redirect")
	}
	return code
}

func extractStateFromLocation(t *testing.T, location string) string {
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("Failed to parse redirect location: %v", err)
	}

	return u.Query().Get("state")
}
