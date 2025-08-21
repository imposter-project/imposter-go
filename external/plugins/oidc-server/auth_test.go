package main

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
)

func createTestOIDCServerForAuth() *OIDCServer {
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

func TestOIDCServer_handleAuthorize(t *testing.T) {
	server := createTestOIDCServerForAuth()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET method",
			method:         "GET",
			expectedStatus: 400, // Will fail due to missing params but shows it routes to GET handler
			expectedBody:   "Missing query parameters",
		},
		{
			name:           "POST method",
			method:         "POST",
			expectedStatus: 400, // Will fail due to missing session_id but shows it routes to POST handler
			expectedBody:   "session_id is required",
		},
		{
			name:           "unsupported method",
			method:         "PUT",
			expectedStatus: 405,
			expectedBody:   "Method Not Allowed",
		},
		{
			name:           "case insensitive method",
			method:         "get",
			expectedStatus: 400,
			expectedBody:   "Missing query parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := shared.HandlerRequest{
				Method: tt.method,
				Path:   "/oidc/authorize",
				Query:  url.Values{},
				Body:   []byte{},
			}

			resp := server.handleAuthorize(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if !strings.Contains(string(resp.Body), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, string(resp.Body))
			}
		})
	}
}

func TestOIDCServer_handleAuthorizeGet(t *testing.T) {
	server := createTestOIDCServerForAuth()

	tests := []struct {
		name           string
		query          url.Values
		expectedStatus int
		checkBody      func(string) bool
	}{
		{
			name:           "missing query parameters",
			query:          url.Values{},
			expectedStatus: 400,
			checkBody: func(body string) bool {
				return strings.Contains(body, "Missing query parameters")
			},
		},
		{
			name: "missing client_id",
			query: url.Values{
				"redirect_uri":  []string{"http://localhost:8080/callback"},
				"response_type": []string{"code"},
				"scope":         []string{"openid"},
			},
			expectedStatus: 400,
			checkBody: func(body string) bool {
				return strings.Contains(body, "client_id is required")
			},
		},
		{
			name: "missing redirect_uri",
			query: url.Values{
				"client_id":     []string{"test-client"},
				"response_type": []string{"code"},
				"scope":         []string{"openid"},
			},
			expectedStatus: 400,
			checkBody: func(body string) bool {
				return strings.Contains(body, "redirect_uri is required")
			},
		},
		{
			name: "unsupported response_type",
			query: url.Values{
				"client_id":     []string{"test-client"},
				"redirect_uri":  []string{"http://localhost:8080/callback"},
				"response_type": []string{"token"}, // Only "code" is supported
				"scope":         []string{"openid"},
			},
			expectedStatus: 302,
			checkBody: func(body string) bool {
				// This is a redirect response, so check the Location header instead
				return true
			},
		},
		{
			name: "missing openid scope",
			query: url.Values{
				"client_id":     []string{"test-client"},
				"redirect_uri":  []string{"http://localhost:8080/callback"},
				"response_type": []string{"code"},
				"scope":         []string{"profile email"}, // Missing "openid"
			},
			expectedStatus: 302,
			checkBody: func(body string) bool {
				// This is a redirect response, so check the Location header instead
				return true
			},
		},
		{
			name: "unknown client",
			query: url.Values{
				"client_id":     []string{"unknown-client"},
				"redirect_uri":  []string{"http://localhost:8080/callback"},
				"response_type": []string{"code"},
				"scope":         []string{"openid"},
			},
			expectedStatus: 302,
			checkBody: func(body string) bool {
				// This is a redirect response, so check the Location header instead
				return true
			},
		},
		{
			name: "invalid redirect_uri",
			query: url.Values{
				"client_id":     []string{"test-client"},
				"redirect_uri":  []string{"http://malicious.com/callback"},
				"response_type": []string{"code"},
				"scope":         []string{"openid"},
			},
			expectedStatus: 400,
			checkBody: func(body string) bool {
				return strings.Contains(body, "Invalid redirect_uri")
			},
		},
		{
			name: "invalid PKCE challenge",
			query: url.Values{
				"client_id":             []string{"test-client"},
				"redirect_uri":          []string{"http://localhost:8080/callback"},
				"response_type":         []string{"code"},
				"scope":                 []string{"openid"},
				"code_challenge":        []string{"short"}, // Too short
				"code_challenge_method": []string{"S256"},
			},
			expectedStatus: 302,
			checkBody: func(body string) bool {
				// This is a redirect response, so check the Location header instead
				return true
			},
		},
		{
			name: "valid request renders login form",
			query: url.Values{
				"client_id":     []string{"test-client"},
				"redirect_uri":  []string{"http://localhost:8080/callback"},
				"response_type": []string{"code"},
				"scope":         []string{"openid profile"},
				"state":         []string{"test-state"},
				"nonce":         []string{"test-nonce"},
			},
			expectedStatus: 200,
			checkBody: func(body string) bool {
				return strings.Contains(body, "Sign In") &&
					strings.Contains(body, "test-client") &&
					strings.Contains(body, "session_id")
			},
		},
		{
			name: "valid request with PKCE",
			query: url.Values{
				"client_id":             []string{"test-client"},
				"redirect_uri":          []string{"http://localhost:8080/callback"},
				"response_type":         []string{"code"},
				"scope":                 []string{"openid"},
				"code_challenge":        []string{"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
				"code_challenge_method": []string{"S256"},
			},
			expectedStatus: 200,
			checkBody: func(body string) bool {
				return strings.Contains(body, "Sign In")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := shared.HandlerRequest{
				Method: "GET",
				Path:   "/oidc/authorize",
				Query:  tt.query,
			}

			resp := server.handleAuthorizeGet(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if !tt.checkBody(string(resp.Body)) {
				t.Errorf("Body check failed for: %s", string(resp.Body))
			}

			// Check that session was created for successful requests
			if tt.expectedStatus == 200 && strings.Contains(tt.name, "valid request") {
				server.mutex.RLock()
				sessionCount := len(server.sessions)
				server.mutex.RUnlock()
				if sessionCount == 0 {
					t.Errorf("Expected at least 1 session to be created, got %d", sessionCount)
				}
			}
		})
	}
}

func TestOIDCServer_handleAuthorizePost(t *testing.T) {
	server := createTestOIDCServerForAuth()

	// Create a test session first
	sessionID := "test-session-id"
	session := &AuthSession{
		ID:          sessionID,
		ClientID:    "test-client",
		RedirectURI: "http://localhost:8080/callback",
		Scope:       "openid profile",
		State:       "test-state",
		Nonce:       "test-nonce",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
	server.sessions[sessionID] = session

	tests := []struct {
		name           string
		body           []byte
		expectedStatus int
		checkResponse  func(*testing.T, shared.HandlerResponse)
	}{
		{
			name:           "invalid form data",
			body:           []byte("invalid form data %"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				if !strings.Contains(string(resp.Body), "Invalid form data") {
					t.Error("Expected invalid form data error")
				}
			},
		},
		{
			name:           "missing session_id",
			body:           []byte("username=alice&password=secret"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				if !strings.Contains(string(resp.Body), "session_id is required") {
					t.Error("Expected session_id required error")
				}
			},
		},
		{
			name:           "invalid session_id",
			body:           []byte("session_id=invalid&username=alice&password=secret"),
			expectedStatus: 400,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				if !strings.Contains(string(resp.Body), "Invalid or expired session") {
					t.Error("Expected invalid session error")
				}
			},
		},
		{
			name:           "invalid credentials",
			body:           []byte("session_id=" + sessionID + "&username=alice&password=wrongpassword"),
			expectedStatus: 200,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				if !strings.Contains(string(resp.Body), "Invalid username or password") {
					t.Error("Expected invalid credentials error in login form")
				}
			},
		},
		{
			name:           "valid credentials",
			body:           []byte("session_id=" + sessionID + "&username=alice&password=password"),
			expectedStatus: 302,
			checkResponse: func(t *testing.T, resp shared.HandlerResponse) {
				location := resp.Headers["Location"]
				if location == "" {
					t.Error("Expected Location header in redirect response")
				}
				if !strings.Contains(location, "code=") {
					t.Error("Expected authorization code in redirect URL")
				}
				if !strings.Contains(location, "state=test-state") {
					t.Error("Expected state parameter in redirect URL")
				}

				// Check that authorization code was created
				server.mutex.RLock()
				codeCount := len(server.codes)
				server.mutex.RUnlock()
				if codeCount != 1 {
					t.Errorf("Expected 1 authorization code to be created, got %d", codeCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset server state for each test (except the session we need)
			server.codes = make(map[string]*AuthCode)
			if tt.name != "invalid session_id" {
				server.sessions = map[string]*AuthSession{sessionID: session}
			} else {
				server.sessions = make(map[string]*AuthSession)
			}

			req := shared.HandlerRequest{
				Method: "POST",
				Path:   "/oidc/authorize",
				Headers: map[string]string{
					"Content-Type": "application/x-www-form-urlencoded",
				},
				Body: tt.body,
			}

			resp := server.handleAuthorizePost(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			tt.checkResponse(t, resp)
		})
	}
}

func TestOIDCServer_renderLoginForm(t *testing.T) {
	server := createTestOIDCServerForAuth()

	tests := []struct {
		name      string
		sessionID string
		clientID  string
		errorMsg  []string
		checkBody func(string) bool
	}{
		{
			name:      "basic login form",
			sessionID: "test-session",
			clientID:  "test-client",
			errorMsg:  []string{},
			checkBody: func(body string) bool {
				return strings.Contains(body, "test-session") &&
					strings.Contains(body, "test-client") &&
					strings.Contains(body, "Sign In") &&
					!strings.Contains(body, "class=\"error\"")
			},
		},
		{
			name:      "login form with error",
			sessionID: "test-session",
			clientID:  "test-client",
			errorMsg:  []string{"Invalid credentials"},
			checkBody: func(body string) bool {
				return strings.Contains(body, "test-session") &&
					strings.Contains(body, "test-client") &&
					strings.Contains(body, "Invalid credentials")
			},
		},
		{
			name:      "no error message",
			sessionID: "session123",
			clientID:  "client456",
			errorMsg:  []string{},
			checkBody: func(body string) bool {
				return strings.Contains(body, "session123") &&
					strings.Contains(body, "client456") &&
					strings.Contains(body, "form") &&
					strings.Contains(body, "username") &&
					strings.Contains(body, "password")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := server.renderLoginForm(tt.sessionID, tt.clientID, tt.errorMsg...)

			if resp.StatusCode != 200 {
				t.Errorf("Expected status code 200, got %d", resp.StatusCode)
			}

			if resp.Headers["Content-Type"] != "text/html; charset=utf-8" {
				t.Errorf("Expected Content-Type text/html; charset=utf-8, got %s", resp.Headers["Content-Type"])
			}

			if !tt.checkBody(string(resp.Body)) {
				t.Errorf("Body check failed for: %s", string(resp.Body))
			}
		})
	}
}

func TestOIDCServer_renderError(t *testing.T) {
	server := createTestOIDCServerForAuth()

	tests := []struct {
		name             string
		errorCode        string
		errorDescription string
	}{
		{
			name:             "basic error",
			errorCode:        "invalid_request",
			errorDescription: "The request is missing a required parameter",
		},
		{
			name:             "server error",
			errorCode:        "server_error",
			errorDescription: "Internal server error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := server.renderError(tt.errorCode, tt.errorDescription)

			if resp.StatusCode != 400 {
				t.Errorf("Expected status code 400, got %d", resp.StatusCode)
			}

			if resp.Headers["Content-Type"] != "text/html; charset=utf-8" {
				t.Errorf("Expected Content-Type text/html; charset=utf-8, got %s", resp.Headers["Content-Type"])
			}

			body := string(resp.Body)
			if !strings.Contains(body, tt.errorCode) {
				t.Errorf("Expected body to contain error code '%s'", tt.errorCode)
			}
			if !strings.Contains(body, tt.errorDescription) {
				t.Errorf("Expected body to contain error description '%s'", tt.errorDescription)
			}
		})
	}
}

func TestOIDCServer_redirectError(t *testing.T) {
	server := createTestOIDCServerForAuth()

	tests := []struct {
		name             string
		redirectURI      string
		errorCode        string
		errorDescription string
		state            string
		expectStatus     int
		checkLocation    func(string) bool
	}{
		{
			name:             "basic redirect error",
			redirectURI:      "http://localhost:8080/callback",
			errorCode:        "access_denied",
			errorDescription: "User denied authorization",
			state:            "test-state",
			expectStatus:     302,
			checkLocation: func(location string) bool {
				return strings.Contains(location, "error=access_denied") &&
					strings.Contains(location, "error_description=User+denied+authorization") &&
					strings.Contains(location, "state=test-state")
			},
		},
		{
			name:             "redirect error without state",
			redirectURI:      "http://localhost:8080/callback",
			errorCode:        "invalid_scope",
			errorDescription: "Requested scope is invalid",
			state:            "",
			expectStatus:     302,
			checkLocation: func(location string) bool {
				return strings.Contains(location, "error=invalid_scope") &&
					strings.Contains(location, "error_description=Requested+scope+is+invalid") &&
					!strings.Contains(location, "state=")
			},
		},
		{
			name:             "invalid redirect URI",
			redirectURI:      "invalid-uri",
			errorCode:        "server_error",
			errorDescription: "Something went wrong",
			state:            "test-state",
			expectStatus:     302, // Still gets redirected even with invalid URI due to url.Parse not failing
			checkLocation: func(location string) bool {
				return strings.Contains(location, "error=server_error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := server.redirectError(tt.redirectURI, tt.errorCode, tt.errorDescription, tt.state)

			if resp.StatusCode != tt.expectStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectStatus, resp.StatusCode)
			}

			location := resp.Headers["Location"]
			if !tt.checkLocation(location) {
				t.Errorf("Location check failed for: %s", location)
			}
		})
	}
}

func TestOIDCServer_AuthFlow_Integration(t *testing.T) {
	server := createTestOIDCServerForAuth()

	t.Run("complete authorization flow", func(t *testing.T) {
		// Step 1: Initial authorization request
		authReq := shared.HandlerRequest{
			Method: "GET",
			Path:   "/oidc/authorize",
			Query: url.Values{
				"client_id":     []string{"test-client"},
				"redirect_uri":  []string{"http://localhost:8080/callback"},
				"response_type": []string{"code"},
				"scope":         []string{"openid profile email"},
				"state":         []string{"integration-test-state"},
				"nonce":         []string{"integration-test-nonce"},
			},
		}

		authResp := server.handleAuthorizeGet(authReq)
		if authResp.StatusCode != 200 {
			t.Fatalf("Expected status 200 for auth request, got %d", authResp.StatusCode)
		}

		// Extract session ID from the form
		body := string(authResp.Body)
		sessionMarker := "name=\"session_id\" value=\""
		sessionStart := strings.Index(body, sessionMarker)
		if sessionStart == -1 {
			t.Fatal("Could not find session_id field in login form")
		}
		sessionStart += len(sessionMarker)
		sessionEnd := strings.Index(body[sessionStart:], "\"")
		if sessionEnd == -1 {
			t.Fatal("Could not find end of session_id value")
		}
		sessionEnd += sessionStart
		sessionID := body[sessionStart:sessionEnd]

		if sessionID == "" {
			t.Fatal("Could not extract session ID from login form")
		}

		// Step 2: User submits valid credentials
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
			t.Fatalf("Expected status 302 for login, got %d", loginResp.StatusCode)
		}

		// Verify redirect contains authorization code
		location := loginResp.Headers["Location"]
		if !strings.Contains(location, "code=") {
			t.Error("Expected authorization code in redirect")
		}
		if !strings.Contains(location, "state=integration-test-state") {
			t.Error("Expected state parameter in redirect")
		}

		// Verify session was cleaned up and code was created
		server.mutex.RLock()
		sessionCount := len(server.sessions)
		codeCount := len(server.codes)
		server.mutex.RUnlock()

		if sessionCount != 0 {
			t.Errorf("Expected session to be cleaned up, got %d sessions", sessionCount)
		}
		if codeCount != 1 {
			t.Errorf("Expected 1 authorization code, got %d", codeCount)
		}
	})
}
