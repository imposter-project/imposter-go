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

func createTestOIDCServerForToken() *OIDCServer {
	return &OIDCServer{
		logger: hclog.New(&hclog.LoggerOptions{
			Level:  hclog.Off,
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

func TestOIDCServer_handleToken(t *testing.T) {
	server := createTestOIDCServerForToken()

	// Create a valid authorization code for testing
	validCode := "valid-auth-code"
	authCode := &AuthCode{
		Code:        validCode,
		ClientID:    "test-client",
		RedirectURI: "http://localhost:8080/callback",
		UserID:      "alice",
		Scope:       "openid profile email",
		Nonce:       "test-nonce",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
	server.codes[validCode] = authCode

	tests := []struct {
		name           string
		method         string
		body           []byte
		expectedStatus int
		checkResponse  func(*testing.T, shared.HandlerResponse)
	}{
		{
			name:           "invalid method GET",
			method:         "GET",
			body:           []byte{},
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				if err := json.Unmarshal(resp.Body, &errorResp); err != nil {
					t.Errorf("Failed to parse error response: %v", err)
				}
				if errorResp.Error != "invalid_request" {
					t.Errorf("Expected error 'invalid_request', got '%s'", errorResp.Error)
				}
			},
		},
		{
			name:           "invalid form data",
			method:         "POST",
			body:           []byte("invalid%form%data"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_request" {
					t.Errorf("Expected error 'invalid_request', got '%s'", errorResp.Error)
				}
			},
		},
		{
			name:           "unsupported grant type",
			method:         "POST",
			body:           []byte("grant_type=client_credentials"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "unsupported_grant_type" {
					t.Errorf("Expected error 'unsupported_grant_type', got '%s'", errorResp.Error)
				}
			},
		},
		{
			name:           "missing code",
			method:         "POST",
			body:           []byte("grant_type=authorization_code&client_id=test-client&redirect_uri=http://localhost:8080/callback"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_request" || !strings.Contains(errorResp.ErrorDescription, "code is required") {
					t.Errorf("Expected code required error, got '%s: %s'", errorResp.Error, errorResp.ErrorDescription)
				}
			},
		},
		{
			name:           "missing client_id",
			method:         "POST",
			body:           []byte("grant_type=authorization_code&code=test-code&redirect_uri=http://localhost:8080/callback"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_client" || !strings.Contains(errorResp.ErrorDescription, "client_id is required") {
					t.Errorf("Expected client_id required error, got '%s: %s'", errorResp.Error, errorResp.ErrorDescription)
				}
			},
		},
		{
			name:           "missing redirect_uri",
			method:         "POST",
			body:           []byte("grant_type=authorization_code&code=test-code&client_id=test-client"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_request" || !strings.Contains(errorResp.ErrorDescription, "redirect_uri is required") {
					t.Errorf("Expected redirect_uri required error, got '%s: %s'", errorResp.Error, errorResp.ErrorDescription)
				}
			},
		},
		{
			name:           "invalid authorization code",
			method:         "POST",
			body:           []byte("grant_type=authorization_code&code=invalid-code&client_id=test-client&redirect_uri=http://localhost:8080/callback"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_grant" {
					t.Errorf("Expected error 'invalid_grant', got '%s'", errorResp.Error)
				}
			},
		},
		{
			name:           "unknown client",
			method:         "POST",
			body:           []byte("grant_type=authorization_code&code=" + validCode + "&client_id=unknown-client&redirect_uri=http://localhost:8080/callback"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_client" {
					t.Errorf("Expected error 'invalid_client', got '%s'", errorResp.Error)
				}
			},
		},
		{
			name:           "redirect_uri mismatch",
			method:         "POST",
			body:           []byte("grant_type=authorization_code&code=" + validCode + "&client_id=test-client&client_secret=test-secret&redirect_uri=http://different.com/callback"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_grant" || !strings.Contains(errorResp.ErrorDescription, "redirect_uri does not match") {
					t.Errorf("Expected redirect_uri mismatch error, got '%s: %s'", errorResp.Error, errorResp.ErrorDescription)
				}
			},
		},
		{
			name:           "valid token request",
			method:         "POST",
			body:           []byte("grant_type=authorization_code&code=" + validCode + "&client_id=test-client&client_secret=test-secret&redirect_uri=http://localhost:8080/callback"),
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var tokenResp TokenResponse
				if err := json.Unmarshal(resp.Body, &tokenResp); err != nil {
					t.Fatalf("Failed to parse token response: %v", err)
				}

				if tokenResp.AccessToken == "" {
					t.Error("Expected access token to be present")
				}
				if tokenResp.TokenType != "Bearer" {
					t.Errorf("Expected token type 'Bearer', got '%s'", tokenResp.TokenType)
				}
				if tokenResp.ExpiresIn != 3600 {
					t.Errorf("Expected expires_in 3600, got %d", tokenResp.ExpiresIn)
				}
				if tokenResp.IDToken == "" {
					t.Error("Expected ID token to be present for openid scope")
				}
				if tokenResp.Scope != "openid profile email" {
					t.Errorf("Expected scope 'openid profile email', got '%s'", tokenResp.Scope)
				}

				// Check that access token was stored
				server.mutex.RLock()
				_, exists := server.tokens[tokenResp.AccessToken]
				server.mutex.RUnlock()
				if !exists {
					t.Error("Access token should be stored in server")
				}

				// Check that authorization code was consumed
				server.mutex.RLock()
				_, exists = server.codes[validCode]
				server.mutex.RUnlock()
				if exists {
					t.Error("Authorization code should be consumed after use")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state for valid token request test
			if tt.name == "valid token request" {
				server.codes = make(map[string]*AuthCode)
				server.tokens = make(map[string]*AccessToken)
				server.codes[validCode] = authCode
			}

			req := shared.HandlerRequest{
				Method: tt.method,
				Path:   "/oidc/token",
				Headers: map[string]string{
					"Content-Type": "application/x-www-form-urlencoded",
				},
				Body: tt.body,
			}

			resp := server.handleToken(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			tt.checkResponse(t, resp)
		})
	}
}

func TestOIDCServer_handleToken_ExpiredCode(t *testing.T) {
	server := createTestOIDCServerForToken()

	// Create an expired authorization code
	expiredCode := "expired-code"
	authCode := &AuthCode{
		Code:        expiredCode,
		ClientID:    "test-client",
		RedirectURI: "http://localhost:8080/callback",
		UserID:      "alice",
		Scope:       "openid",
		ExpiresAt:   time.Now().Add(-1 * time.Hour), // Expired
	}
	server.codes[expiredCode] = authCode

	req := shared.HandlerRequest{
		Method: "POST",
		Path:   "/oidc/token",
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		Body: []byte("grant_type=authorization_code&code=" + expiredCode + "&client_id=test-client&redirect_uri=http://localhost:8080/callback"),
	}

	resp := server.handleToken(req)

	if resp.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", resp.StatusCode)
	}

	var errorResp TokenErrorResponse
	if err := json.Unmarshal(resp.Body, &errorResp); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if errorResp.Error != "invalid_grant" || !strings.Contains(errorResp.ErrorDescription, "expired") {
		t.Errorf("Expected expired code error, got '%s: %s'", errorResp.Error, errorResp.ErrorDescription)
	}

	// Check that expired code was cleaned up
	server.mutex.RLock()
	_, exists := server.codes[expiredCode]
	server.mutex.RUnlock()
	if exists {
		t.Error("Expired code should be removed from storage")
	}
}

func TestOIDCServer_handleToken_PKCE(t *testing.T) {
	server := createTestOIDCServerForToken()

	// Create authorization code with PKCE
	codeWithPKCE := "code-with-pkce"
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" // S256 challenge
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"  // Corresponding verifier

	authCode := &AuthCode{
		Code:            codeWithPKCE,
		ClientID:        "test-client",
		RedirectURI:     "http://localhost:8080/callback",
		UserID:          "alice",
		Scope:           "openid",
		CodeChallenge:   challenge,
		ChallengeMethod: "S256",
		ExpiresAt:       time.Now().Add(10 * time.Minute),
	}
	server.codes[codeWithPKCE] = authCode

	tests := []struct {
		name         string
		codeVerifier string
		expectError  bool
	}{
		{
			name:         "valid PKCE verifier",
			codeVerifier: verifier,
			expectError:  false,
		},
		{
			name:         "missing code verifier",
			codeVerifier: "",
			expectError:  true,
		},
		{
			name:         "invalid code verifier",
			codeVerifier: "invalid-verifier",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state
			server.codes = make(map[string]*AuthCode)
			server.tokens = make(map[string]*AccessToken)
			server.codes[codeWithPKCE] = authCode

			body := "grant_type=authorization_code&code=" + codeWithPKCE + "&client_id=test-client&client_secret=test-secret&redirect_uri=http://localhost:8080/callback"
			if tt.codeVerifier != "" {
				body += "&code_verifier=" + url.QueryEscape(tt.codeVerifier)
			}

			req := shared.HandlerRequest{
				Method: "POST",
				Path:   "/oidc/token",
				Headers: map[string]string{
					"Content-Type": "application/x-www-form-urlencoded",
				},
				Body: []byte(body),
			}

			resp := server.handleToken(req)

			if tt.expectError {
				if resp.StatusCode != 400 {
					t.Errorf("Expected status code 400, got %d", resp.StatusCode)
				}
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_grant" {
					t.Errorf("Expected error 'invalid_grant', got '%s'", errorResp.Error)
				}
			} else {
				if resp.StatusCode != 200 {
					t.Errorf("Expected status code 200, got %d", resp.StatusCode)
				}
			}
		})
	}
}

func TestOIDCServer_generateIDToken(t *testing.T) {
	server := createTestOIDCServerForToken()
	server.serverURL = "https://example.com"

	user := &User{
		Username: "testuser",
		Claims: map[string]string{
			"sub":         "testuser",
			"email":       "test@example.com",
			"given_name":  "Test",
			"family_name": "User",
		},
	}

	issuedAt := time.Now()
	expiresIn := 3600

	tests := []struct {
		name     string
		clientID string
		nonce    string
		scopes   []string
		validate func(*testing.T, string)
	}{
		{
			name:     "basic ID token",
			clientID: "test-client",
			nonce:    "",
			scopes:   []string{"openid"},
			validate: func(t *testing.T, tokenStr string) {
				token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
					return server.jwtSecret, nil
				})
				if err != nil {
					t.Fatalf("Failed to parse JWT: %v", err)
				}

				claims := token.Claims.(jwt.MapClaims)
				if claims["iss"] != "https://example.com" {
					t.Errorf("Expected iss 'https://example.com', got '%s'", claims["iss"])
				}
				if claims["sub"] != "testuser" {
					t.Errorf("Expected sub 'testuser', got '%s'", claims["sub"])
				}
				if claims["aud"] != "test-client" {
					t.Errorf("Expected aud 'test-client', got '%s'", claims["aud"])
				}
				if claims["nonce"] != nil {
					t.Error("Expected no nonce claim")
				}
			},
		},
		{
			name:     "ID token with nonce",
			clientID: "test-client",
			nonce:    "test-nonce-123",
			scopes:   []string{"openid"},
			validate: func(t *testing.T, tokenStr string) {
				token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
					return server.jwtSecret, nil
				})
				if err != nil {
					t.Fatalf("Failed to parse JWT: %v", err)
				}

				claims := token.Claims.(jwt.MapClaims)
				if claims["nonce"] != "test-nonce-123" {
					t.Errorf("Expected nonce 'test-nonce-123', got '%s'", claims["nonce"])
				}
			},
		},
		{
			name:     "ID token with profile claims",
			clientID: "test-client",
			nonce:    "",
			scopes:   []string{"openid", "profile", "email"},
			validate: func(t *testing.T, tokenStr string) {
				token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
					return server.jwtSecret, nil
				})
				if err != nil {
					t.Fatalf("Failed to parse JWT: %v", err)
				}

				claims := token.Claims.(jwt.MapClaims)
				if claims["email"] != "test@example.com" {
					t.Errorf("Expected email 'test@example.com', got '%s'", claims["email"])
				}
				if claims["given_name"] != "Test" {
					t.Errorf("Expected given_name 'Test', got '%s'", claims["given_name"])
				}
				if claims["family_name"] != "User" {
					t.Errorf("Expected family_name 'User', got '%s'", claims["family_name"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idToken, err := server.generateIDToken(user, tt.clientID, tt.nonce, tt.scopes, issuedAt, expiresIn)
			if err != nil {
				t.Fatalf("Failed to generate ID token: %v", err)
			}

			if idToken == "" {
				t.Error("ID token should not be empty")
			}

			tt.validate(t, idToken)
		})
	}
}

func TestOIDCServer_validateAccessToken(t *testing.T) {
	server := createTestOIDCServerForToken()

	validToken := "valid-access-token"
	expiredToken := "expired-access-token"

	// Add valid token
	server.tokens[validToken] = &AccessToken{
		Token:     validToken,
		UserID:    "alice",
		ClientID:  "test-client",
		Scope:     "openid profile",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Add expired token
	server.tokens[expiredToken] = &AccessToken{
		Token:     expiredToken,
		UserID:    "bob",
		ClientID:  "test-client",
		Scope:     "openid",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}

	tests := []struct {
		name        string
		token       string
		expectError bool
		checkResult func(*testing.T, *AccessToken, error)
	}{
		{
			name:        "valid token",
			token:       validToken,
			expectError: false,
			checkResult: func(t *testing.T, token *AccessToken, err error) {
				if token == nil {
					t.Error("Expected token to be returned")
				}
				if token.UserID != "alice" {
					t.Errorf("Expected UserID 'alice', got '%s'", token.UserID)
				}
			},
		},
		{
			name:        "invalid token",
			token:       "nonexistent-token",
			expectError: true,
			checkResult: func(t *testing.T, token *AccessToken, err error) {
				if !strings.Contains(err.Error(), "invalid token") {
					t.Errorf("Expected 'invalid token' error, got '%v'", err)
				}
			},
		},
		{
			name:        "expired token",
			token:       expiredToken,
			expectError: true,
			checkResult: func(t *testing.T, token *AccessToken, err error) {
				if !strings.Contains(err.Error(), "token expired") {
					t.Errorf("Expected 'token expired' error, got '%v'", err)
				}
				// Check that expired token was cleaned up
				server.mutex.RLock()
				_, exists := server.tokens[expiredToken]
				server.mutex.RUnlock()
				if exists {
					t.Error("Expired token should be removed from storage")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := server.validateAccessToken(tt.token)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			tt.checkResult(t, token, err)
		})
	}
}

func TestOIDCServer_tokenError(t *testing.T) {
	server := createTestOIDCServerForToken()

	resp := server.tokenError("invalid_grant", "Authorization code has expired")

	if resp.StatusCode != 400 {
		t.Errorf("Expected status code 400, got %d", resp.StatusCode)
	}

	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Headers["Content-Type"])
	}

	if resp.Headers["Cache-Control"] != "no-store" {
		t.Errorf("Expected Cache-Control no-store, got %s", resp.Headers["Cache-Control"])
	}

	var errorResp TokenErrorResponse
	if err := json.Unmarshal(resp.Body, &errorResp); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if errorResp.Error != "invalid_grant" {
		t.Errorf("Expected error 'invalid_grant', got '%s'", errorResp.Error)
	}
	if errorResp.ErrorDescription != "Authorization code has expired" {
		t.Errorf("Expected error description 'Authorization code has expired', got '%s'", errorResp.ErrorDescription)
	}
}

func TestContainsScope(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		scope  string
		want   bool
	}{
		{
			name:   "scope exists",
			scopes: []string{"openid", "profile", "email"},
			scope:  "profile",
			want:   true,
		},
		{
			name:   "scope does not exist",
			scopes: []string{"openid", "profile"},
			scope:  "email",
			want:   false,
		},
		{
			name:   "empty scopes",
			scopes: []string{},
			scope:  "openid",
			want:   false,
		},
		{
			name:   "single scope match",
			scopes: []string{"openid"},
			scope:  "openid",
			want:   true,
		},
		{
			name:   "case sensitive",
			scopes: []string{"OpenID"},
			scope:  "openid",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsScope(tt.scopes, tt.scope)
			if got != tt.want {
				t.Errorf("containsScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBasicAuth(t *testing.T) {
	tests := []struct {
		name           string
		header         string
		expectUsername string
		expectPassword string
		expectOk       bool
	}{
		{
			name:           "not basic auth",
			header:         "Bearer token123",
			expectUsername: "",
			expectPassword: "",
			expectOk:       false,
		},
		{
			name:           "empty header",
			header:         "",
			expectUsername: "",
			expectPassword: "",
			expectOk:       false,
		},
		{
			name:           "basic prefix only",
			header:         "Basic",
			expectUsername: "",
			expectPassword: "",
			expectOk:       false,
		},
		{
			name:           "valid basic auth",
			header:         "Basic dGVzdDpzZWNyZXQ=", // test:secret in base64
			expectUsername: "test",
			expectPassword: "secret",
			expectOk:       true,
		},
		{
			name:           "basic auth with empty password",
			header:         "Basic dGVzdDo=", // test: in base64
			expectUsername: "test",
			expectPassword: "",
			expectOk:       true,
		},
		{
			name:           "basic auth with colon in password",
			header:         "Basic Y2xpZW50MTpzZWNyZXQ6d2l0aDpjb2xvbnM=", // client1:secret:with:colons in base64
			expectUsername: "client1",
			expectPassword: "secret:with:colons",
			expectOk:       true,
		},
		{
			name:           "invalid base64",
			header:         "Basic invalid_base64!",
			expectUsername: "",
			expectPassword: "",
			expectOk:       false,
		},
		{
			name:           "missing colon separator",
			header:         "Basic dGVzdA==", // test in base64 (no colon)
			expectUsername: "",
			expectPassword: "",
			expectOk:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, password, ok := parseBasicAuth(tt.header)

			if username != tt.expectUsername {
				t.Errorf("Expected username '%s', got '%s'", tt.expectUsername, username)
			}
			if password != tt.expectPassword {
				t.Errorf("Expected password '%s', got '%s'", tt.expectPassword, password)
			}
			if ok != tt.expectOk {
				t.Errorf("Expected ok %v, got %v", tt.expectOk, ok)
			}
		})
	}
}

func TestOIDCServer_TokenFlow_Integration(t *testing.T) {
	server := createTestOIDCServerForToken()

	t.Run("complete token exchange flow", func(t *testing.T) {
		// Setup: Create authorization code
		authCode := &AuthCode{
			Code:        "integration-test-code",
			ClientID:    "test-client",
			RedirectURI: "http://localhost:8080/callback",
			UserID:      "alice",
			Scope:       "openid profile email",
			Nonce:       "integration-nonce",
			ExpiresAt:   time.Now().Add(10 * time.Minute),
		}
		server.codes[authCode.Code] = authCode

		// Execute: Exchange code for tokens
		req := shared.HandlerRequest{
			Method: "POST",
			Path:   "/oidc/token",
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
			Body: []byte("grant_type=authorization_code&code=integration-test-code&client_id=test-client&client_secret=test-secret&redirect_uri=http://localhost:8080/callback"),
		}

		resp := server.handleToken(req)

		// Verify: Token response
		if resp.StatusCode != 200 {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var tokenResp TokenResponse
		if err := json.Unmarshal(resp.Body, &tokenResp); err != nil {
			t.Fatalf("Failed to parse token response: %v", err)
		}

		if tokenResp.AccessToken == "" {
			t.Error("Expected access token")
		}
		if tokenResp.IDToken == "" {
			t.Error("Expected ID token")
		}
		if tokenResp.TokenType != "Bearer" {
			t.Error("Expected Bearer token type")
		}

		// Verify: Access token is stored and valid
		storedToken, err := server.validateAccessToken(tokenResp.AccessToken)
		if err != nil {
			t.Errorf("Access token validation failed: %v", err)
		}
		if storedToken.UserID != "alice" {
			t.Errorf("Expected UserID 'alice', got '%s'", storedToken.UserID)
		}

		// Verify: ID token is valid JWT
		token, err := jwt.Parse(tokenResp.IDToken, func(token *jwt.Token) (interface{}, error) {
			return server.jwtSecret, nil
		})
		if err != nil {
			t.Errorf("ID token validation failed: %v", err)
		}

		claims := token.Claims.(jwt.MapClaims)
		if claims["sub"] != "alice" {
			t.Errorf("Expected sub 'alice' in ID token, got '%s'", claims["sub"])
		}
		if claims["nonce"] != "integration-nonce" {
			t.Errorf("Expected nonce in ID token")
		}

		// Verify: Authorization code was consumed
		server.mutex.RLock()
		_, exists := server.codes[authCode.Code]
		server.mutex.RUnlock()
		if exists {
			t.Error("Authorization code should be consumed")
		}
	})
}

func TestOIDCServer_handleToken_ClientSecretBasic(t *testing.T) {
	server := createTestOIDCServerForToken()

	// Create a valid authorization code for testing
	validCode := "valid-auth-code-basic"
	authCode := &AuthCode{
		Code:        validCode,
		ClientID:    "test-client",
		RedirectURI: "http://localhost:8080/callback",
		UserID:      "alice",
		Scope:       "openid profile",
		Nonce:       "test-nonce",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
	server.codes[validCode] = authCode

	tests := []struct {
		name                string
		authorizationHeader string
		body                string
		expectedStatus      int
		checkResponse       func(*testing.T, shared.HandlerResponse)
	}{
		{
			name:                "valid client_secret_basic authentication",
			authorizationHeader: "Basic dGVzdC1jbGllbnQ6dGVzdC1zZWNyZXQ=", // test-client:test-secret in base64
			body:                "grant_type=authorization_code&code=" + validCode + "&redirect_uri=http://localhost:8080/callback",
			expectedStatus:      200,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var tokenResp TokenResponse
				if err := json.Unmarshal(resp.Body, &tokenResp); err != nil {
					t.Fatalf("Failed to parse token response: %v", err)
				}
				if tokenResp.AccessToken == "" {
					t.Error("Expected access token to be present")
				}
				if tokenResp.TokenType != "Bearer" {
					t.Errorf("Expected token type 'Bearer', got '%s'", tokenResp.TokenType)
				}
			},
		},
		{
			name:                "invalid client credentials in Authorization header",
			authorizationHeader: "Basic dGVzdC1jbGllbnQ6d3Jvbmctc2VjcmV0", // test-client:wrong-secret in base64
			body:                "grant_type=authorization_code&code=" + validCode + "&redirect_uri=http://localhost:8080/callback",
			expectedStatus:      400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_client" {
					t.Errorf("Expected error 'invalid_client', got '%s'", errorResp.Error)
				}
			},
		},
		{
			name:                "unknown client in Authorization header",
			authorizationHeader: "Basic dW5rbm93bi1jbGllbnQ6c2VjcmV0", // unknown-client:secret in base64
			body:                "grant_type=authorization_code&code=" + validCode + "&redirect_uri=http://localhost:8080/callback",
			expectedStatus:      400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_client" {
					t.Errorf("Expected error 'invalid_client', got '%s'", errorResp.Error)
				}
			},
		},
		{
			name:                "malformed Authorization header",
			authorizationHeader: "Basic invalid_base64!@#",
			body:                "grant_type=authorization_code&code=" + validCode + "&redirect_uri=http://localhost:8080/callback",
			expectedStatus:      400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp TokenErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_client" {
					t.Errorf("Expected error 'invalid_client', got '%s'", errorResp.Error)
				}
			},
		},
		{
			name:                "client_secret_post takes precedence when both methods present",
			authorizationHeader: "Basic dGVzdC1jbGllbnQ6d3Jvbmctc2VjcmV0", // wrong credentials in header
			body:                "grant_type=authorization_code&code=" + validCode + "&redirect_uri=http://localhost:8080/callback&client_id=test-client&client_secret=test-secret",
			expectedStatus:      200, // Should succeed using form data credentials
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var tokenResp TokenResponse
				if err := json.Unmarshal(resp.Body, &tokenResp); err != nil {
					t.Fatalf("Failed to parse token response: %v", err)
				}
				if tokenResp.AccessToken == "" {
					t.Error("Expected access token to be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset state for each test
			server.codes = make(map[string]*AuthCode)
			server.tokens = make(map[string]*AccessToken)
			server.codes[validCode] = authCode

			headers := map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			}
			if tt.authorizationHeader != "" {
				headers["Authorization"] = tt.authorizationHeader
			}

			req := shared.HandlerRequest{
				Method:  "POST",
				Path:    "/oidc/token",
				Headers: headers,
				Body:    []byte(tt.body),
			}

			resp := server.handleToken(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			tt.checkResponse(t, resp)
		})
	}
}
