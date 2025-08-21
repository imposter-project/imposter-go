package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
)

func createTestOIDCServerForUserinfo() *OIDCServer {
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

func TestOIDCServer_handleUserInfo(t *testing.T) {
	server := createTestOIDCServerForUserinfo()

	// Create a valid access token
	validToken := "valid-access-token"
	server.tokens[validToken] = &AccessToken{
		Token:     validToken,
		UserID:    "alice",
		ClientID:  "test-client",
		Scope:     "openid profile email",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	tests := []struct {
		name           string
		method         string
		headers        map[string]string
		expectedStatus int
		checkResponse  func(*testing.T, shared.HandlerResponse)
	}{
		{
			name:           "invalid method",
			method:         "PUT",
			headers:        map[string]string{},
			expectedStatus: 405,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp UserinfoErrorResponse
				if err := json.Unmarshal(resp.Body, &errorResp); err != nil {
					t.Errorf("Failed to parse error response: %v", err)
				}
				if errorResp.Error != "invalid_request" {
					t.Errorf("Expected error 'invalid_request', got '%s'", errorResp.Error)
				}
			},
		},
		{
			name:   "missing authorization header",
			method: "GET",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			expectedStatus: 401,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp UserinfoErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_token" {
					t.Errorf("Expected error 'invalid_token', got '%s'", errorResp.Error)
				}
				if !strings.Contains(errorResp.ErrorDescription, "Authorization header is required") {
					t.Error("Expected authorization header required error")
				}
				// Check WWW-Authenticate header
				if resp.Headers["WWW-Authenticate"] != `Bearer realm="oidc-server"` {
					t.Errorf("Expected WWW-Authenticate header, got '%s'", resp.Headers["WWW-Authenticate"])
				}
			},
		},
		{
			name:   "invalid authorization header format",
			method: "GET",
			headers: map[string]string{
				"Authorization": "Basic dGVzdA==",
			},
			expectedStatus: 401,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp UserinfoErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_token" {
					t.Errorf("Expected error 'invalid_token', got '%s'", errorResp.Error)
				}
				if !strings.Contains(errorResp.ErrorDescription, "Invalid authorization header format") {
					t.Error("Expected invalid header format error")
				}
			},
		},
		{
			name:   "empty bearer token",
			method: "GET",
			headers: map[string]string{
				"Authorization": "Bearer ",
			},
			expectedStatus: 401,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp UserinfoErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_token" {
					t.Errorf("Expected error 'invalid_token', got '%s'", errorResp.Error)
				}
				if !strings.Contains(errorResp.ErrorDescription, "Access token is required") {
					t.Error("Expected access token required error")
				}
			},
		},
		{
			name:   "invalid access token",
			method: "GET",
			headers: map[string]string{
				"Authorization": "Bearer invalid-token",
			},
			expectedStatus: 401,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var errorResp UserinfoErrorResponse
				json.Unmarshal(resp.Body, &errorResp)
				if errorResp.Error != "invalid_token" {
					t.Errorf("Expected error 'invalid_token', got '%s'", errorResp.Error)
				}
			},
		},
		{
			name:   "case insensitive authorization header - lowercase",
			method: "GET",
			headers: map[string]string{
				"authorization": "Bearer " + validToken,
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var userClaims map[string]interface{}
				if err := json.Unmarshal(resp.Body, &userClaims); err != nil {
					t.Errorf("Failed to parse userinfo response: %v", err)
				}
				if userClaims["sub"] != "alice" {
					t.Errorf("Expected sub 'alice', got '%s'", userClaims["sub"])
				}
			},
		},
		{
			name:   "valid GET request",
			method: "GET",
			headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				if resp.Headers["Content-Type"] != "application/json" {
					t.Errorf("Expected Content-Type application/json, got '%s'", resp.Headers["Content-Type"])
				}
				if resp.Headers["Cache-Control"] != "no-store" {
					t.Errorf("Expected Cache-Control no-store, got '%s'", resp.Headers["Cache-Control"])
				}

				var userClaims map[string]interface{}
				if err := json.Unmarshal(resp.Body, &userClaims); err != nil {
					t.Errorf("Failed to parse userinfo response: %v", err)
				}

				// Check that user claims are included based on scope
				if userClaims["sub"] != "alice" {
					t.Errorf("Expected sub 'alice', got '%s'", userClaims["sub"])
				}
				if userClaims["email"] == nil {
					t.Error("Expected email claim for email scope")
				}
				if userClaims["given_name"] == nil {
					t.Error("Expected given_name claim for profile scope")
				}
			},
		},
		{
			name:   "valid POST request",
			method: "POST",
			headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				var userClaims map[string]interface{}
				if err := json.Unmarshal(resp.Body, &userClaims); err != nil {
					t.Errorf("Failed to parse userinfo response: %v", err)
				}
				if userClaims["sub"] != "alice" {
					t.Errorf("Expected sub 'alice', got '%s'", userClaims["sub"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := shared.HandlerRequest{
				Method:  tt.method,
				Path:    "/oidc/userinfo",
				Headers: tt.headers,
			}

			resp := server.handleUserInfo(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			tt.checkResponse(t, resp)
		})
	}
}

func TestOIDCServer_handleUserInfo_ExpiredToken(t *testing.T) {
	server := createTestOIDCServerForUserinfo()

	// Create an expired access token
	expiredToken := "expired-access-token"
	server.tokens[expiredToken] = &AccessToken{
		Token:     expiredToken,
		UserID:    "alice",
		ClientID:  "test-client",
		Scope:     "openid",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}

	req := shared.HandlerRequest{
		Method: "GET",
		Path:   "/oidc/userinfo",
		Headers: map[string]string{
			"Authorization": "Bearer " + expiredToken,
		},
	}

	resp := server.handleUserInfo(req)

	if resp.StatusCode != 401 {
		t.Errorf("Expected status code 401, got %d", resp.StatusCode)
	}

	var errorResp UserinfoErrorResponse
	if err := json.Unmarshal(resp.Body, &errorResp); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if errorResp.Error != "invalid_token" {
		t.Errorf("Expected error 'invalid_token', got '%s'", errorResp.Error)
	}
	if !strings.Contains(errorResp.ErrorDescription, "token expired") {
		t.Error("Expected token expired error message")
	}

	// Check that expired token was cleaned up
	server.mutex.RLock()
	_, exists := server.tokens[expiredToken]
	server.mutex.RUnlock()
	if exists {
		t.Error("Expired token should be removed from storage")
	}
}

func TestOIDCServer_handleUserInfo_ScopeBasedClaims(t *testing.T) {
	server := createTestOIDCServerForUserinfo()

	tests := []struct {
		name        string
		scope       string
		checkClaims func(*testing.T, map[string]interface{})
	}{
		{
			name:  "openid scope only",
			scope: "openid",
			checkClaims: func(t *testing.T, claims map[string]interface{}) {
				if claims["sub"] != "alice" {
					t.Error("Expected sub claim")
				}
				if claims["email"] != nil {
					t.Error("Email claim should not be present for openid-only scope")
				}
				if claims["given_name"] != nil {
					t.Error("Profile claims should not be present for openid-only scope")
				}
			},
		},
		{
			name:  "openid and profile scopes",
			scope: "openid profile",
			checkClaims: func(t *testing.T, claims map[string]interface{}) {
				if claims["sub"] != "alice" {
					t.Error("Expected sub claim")
				}
				if claims["given_name"] == nil {
					t.Error("Expected given_name claim for profile scope")
				}
				if claims["family_name"] == nil {
					t.Error("Expected family_name claim for profile scope")
				}
				if claims["email"] != nil {
					t.Error("Email claim should not be present without email scope")
				}
			},
		},
		{
			name:  "openid and email scopes",
			scope: "openid email",
			checkClaims: func(t *testing.T, claims map[string]interface{}) {
				if claims["sub"] != "alice" {
					t.Error("Expected sub claim")
				}
				if claims["email"] == nil {
					t.Error("Expected email claim for email scope")
				}
				if claims["given_name"] != nil {
					t.Error("Profile claims should not be present without profile scope")
				}
			},
		},
		{
			name:  "all standard scopes",
			scope: "openid profile email",
			checkClaims: func(t *testing.T, claims map[string]interface{}) {
				if claims["sub"] != "alice" {
					t.Error("Expected sub claim")
				}
				if claims["email"] == nil {
					t.Error("Expected email claim")
				}
				if claims["given_name"] == nil {
					t.Error("Expected given_name claim")
				}
				if claims["family_name"] == nil {
					t.Error("Expected family_name claim")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenName := "token-" + strings.ReplaceAll(tt.name, " ", "-")
			server.tokens[tokenName] = &AccessToken{
				Token:     tokenName,
				UserID:    "alice",
				ClientID:  "test-client",
				Scope:     tt.scope,
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}

			req := shared.HandlerRequest{
				Method: "GET",
				Path:   "/oidc/userinfo",
				Headers: map[string]string{
					"Authorization": "Bearer " + tokenName,
				},
			}

			resp := server.handleUserInfo(req)

			if resp.StatusCode != 200 {
				t.Fatalf("Expected status 200, got %d", resp.StatusCode)
			}

			var userClaims map[string]interface{}
			if err := json.Unmarshal(resp.Body, &userClaims); err != nil {
				t.Fatalf("Failed to parse userinfo response: %v", err)
			}

			tt.checkClaims(t, userClaims)
		})
	}
}

func TestOIDCServer_handleUserInfo_UserNotFound(t *testing.T) {
	server := createTestOIDCServerForUserinfo()

	// Create access token for non-existent user
	token := "token-for-missing-user"
	server.tokens[token] = &AccessToken{
		Token:     token,
		UserID:    "nonexistent-user",
		ClientID:  "test-client",
		Scope:     "openid",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	req := shared.HandlerRequest{
		Method: "GET",
		Path:   "/oidc/userinfo",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
	}

	resp := server.handleUserInfo(req)

	if resp.StatusCode != 500 {
		t.Errorf("Expected status code 500, got %d", resp.StatusCode)
	}

	var errorResp UserinfoErrorResponse
	if err := json.Unmarshal(resp.Body, &errorResp); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if errorResp.Error != "server_error" {
		t.Errorf("Expected error 'server_error', got '%s'", errorResp.Error)
	}
	if !strings.Contains(errorResp.ErrorDescription, "User not found") {
		t.Error("Expected user not found error message")
	}
}

func TestOIDCServer_handleUserInfo_MissingSubClaim(t *testing.T) {
	server := createTestOIDCServerForUserinfo()

	// Create a custom user without sub claim in config
	customUser := &User{
		Username: "testuser",
		Password: "testpass",
		Claims:   map[string]string{}, // No sub claim
	}
	server.config.Users = append(server.config.Users, *customUser)

	token := "token-no-sub"
	server.tokens[token] = &AccessToken{
		Token:     token,
		UserID:    "testuser",
		ClientID:  "test-client",
		Scope:     "openid",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	req := shared.HandlerRequest{
		Method: "GET",
		Path:   "/oidc/userinfo",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
	}

	resp := server.handleUserInfo(req)

	if resp.StatusCode != 200 {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var userClaims map[string]interface{}
	if err := json.Unmarshal(resp.Body, &userClaims); err != nil {
		t.Fatalf("Failed to parse userinfo response: %v", err)
	}

	// Should default sub to username
	if userClaims["sub"] != "testuser" {
		t.Errorf("Expected sub to default to username 'testuser', got '%s'", userClaims["sub"])
	}
}

func TestOIDCServer_userinfoError(t *testing.T) {
	server := createTestOIDCServerForUserinfo()

	tests := []struct {
		name         string
		errorCode    string
		errorDesc    string
		statusCode   int
		checkHeaders func(*testing.T, map[string]string)
	}{
		{
			name:       "401 error with WWW-Authenticate",
			errorCode:  "invalid_token",
			errorDesc:  "Token has expired",
			statusCode: 401,
			checkHeaders: func(t *testing.T, headers map[string]string) {
				if headers["WWW-Authenticate"] != `Bearer realm="oidc-server"` {
					t.Errorf("Expected WWW-Authenticate header, got '%s'", headers["WWW-Authenticate"])
				}
				if headers["Content-Type"] != "application/json" {
					t.Errorf("Expected Content-Type application/json, got '%s'", headers["Content-Type"])
				}
			},
		},
		{
			name:       "500 error without WWW-Authenticate",
			errorCode:  "server_error",
			errorDesc:  "Internal server error",
			statusCode: 500,
			checkHeaders: func(t *testing.T, headers map[string]string) {
				if _, exists := headers["WWW-Authenticate"]; exists {
					t.Error("WWW-Authenticate header should not be present for non-401 errors")
				}
				if headers["Content-Type"] != "application/json" {
					t.Errorf("Expected Content-Type application/json, got '%s'", headers["Content-Type"])
				}
			},
		},
		{
			name:       "400 error",
			errorCode:  "invalid_request",
			errorDesc:  "Bad request",
			statusCode: 400,
			checkHeaders: func(t *testing.T, headers map[string]string) {
				if _, exists := headers["WWW-Authenticate"]; exists {
					t.Error("WWW-Authenticate header should not be present for non-401 errors")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := server.userinfoError(tt.errorCode, tt.errorDesc, tt.statusCode)

			if resp.StatusCode != tt.statusCode {
				t.Errorf("Expected status code %d, got %d", tt.statusCode, resp.StatusCode)
			}

			var errorResp UserinfoErrorResponse
			if err := json.Unmarshal(resp.Body, &errorResp); err != nil {
				t.Fatalf("Failed to parse error response: %v", err)
			}

			if errorResp.Error != tt.errorCode {
				t.Errorf("Expected error '%s', got '%s'", tt.errorCode, errorResp.Error)
			}
			if errorResp.ErrorDescription != tt.errorDesc {
				t.Errorf("Expected error description '%s', got '%s'", tt.errorDesc, errorResp.ErrorDescription)
			}

			tt.checkHeaders(t, resp.Headers)
		})
	}
}

func TestOIDCServer_UserInfoFlow_Integration(t *testing.T) {
	server := createTestOIDCServerForUserinfo()

	t.Run("complete userinfo flow", func(t *testing.T) {
		// Setup: Create access token with comprehensive scope
		accessToken := "integration-access-token"
		server.tokens[accessToken] = &AccessToken{
			Token:     accessToken,
			UserID:    "alice",
			ClientID:  "test-client",
			Scope:     "openid profile email",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		// Execute: Request userinfo
		req := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/userinfo",
			Headers: map[string]string{
				"Authorization": "Bearer " + accessToken,
			},
		}

		resp := server.handleUserInfo(req)

		// Verify: Userinfo response
		if resp.StatusCode != 200 {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		var userClaims map[string]interface{}
		if err := json.Unmarshal(resp.Body, &userClaims); err != nil {
			t.Fatalf("Failed to parse userinfo response: %v", err)
		}

		// Verify: Expected claims are present
		expectedClaims := []string{"sub", "email", "given_name", "family_name", "name"}
		for _, claim := range expectedClaims {
			if userClaims[claim] == nil {
				t.Errorf("Expected claim '%s' to be present", claim)
			}
		}

		// Verify: Sub matches user
		if userClaims["sub"] != "alice" {
			t.Errorf("Expected sub 'alice', got '%s'", userClaims["sub"])
		}

		// Verify: Response headers
		if resp.Headers["Content-Type"] != "application/json" {
			t.Error("Expected JSON content type")
		}
		if resp.Headers["Cache-Control"] != "no-store" {
			t.Error("Expected no-store cache control")
		}
	})
}
