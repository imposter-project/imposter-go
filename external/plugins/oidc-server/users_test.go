package main

import (
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/crypto/bcrypt"
)

func createTestOIDCServer() *OIDCServer {
	return &OIDCServer{
		logger: hclog.New(&hclog.LoggerOptions{
			Level:      hclog.Trace,
			Output:     os.Stderr,
			JSONFormat: true,
		}),
		config: &OIDCConfig{
			Users: []User{
				{
					Username: "alice",
					Password: "plaintext-password",
					Claims: map[string]string{
						"sub":         "alice",
						"email":       "alice@example.com",
						"given_name":  "Alice",
						"family_name": "Smith",
						"name":        "Alice Smith",
					},
				},
				{
					Username: "bob",
					Password: "$2a$10$DkpuQ/LhGg/ZnEZW3i1SReY9MkQr4kFpvXNy9uG0B8sXUOID8TtCa", // bcrypt hash of "secret"
					Claims: map[string]string{
						"sub":         "bob",
						"email":       "bob@example.com",
						"given_name":  "Bob",
						"family_name": "Jones",
					},
				},
				{
					Username: "charlie",
					Password: "charlie-pass",
					Claims: map[string]string{
						"sub":                   "charlie",
						"email":                 "charlie@example.com",
						"email_verified":        "true",
						"phone_number":          "+1234567890",
						"phone_number_verified": "false",
						"address":               "123 Main St",
						"nickname":              "Chuck",
						"preferred_username":    "charles",
					},
				},
			},
		},
	}
}

func TestOIDCServer_authenticateUser(t *testing.T) {
	server := createTestOIDCServer()

	tests := []struct {
		name     string
		username string
		password string
		want     *User
	}{
		{
			name:     "valid plaintext password",
			username: "alice",
			password: "plaintext-password",
			want:     &server.config.Users[0],
		},
		{
			name:     "valid bcrypt password",
			username: "bob",
			password: "secret",
			want:     &server.config.Users[1],
		},
		{
			name:     "invalid password",
			username: "alice",
			password: "wrong-password",
			want:     nil,
		},
		{
			name:     "non-existent user",
			username: "nonexistent",
			password: "any-password",
			want:     nil,
		},
		{
			name:     "empty username",
			username: "",
			password: "any-password",
			want:     nil,
		},
		{
			name:     "empty password",
			username: "alice",
			password: "",
			want:     nil,
		},
		{
			name:     "case sensitive username",
			username: "Alice", // Different case
			password: "plaintext-password",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := server.authenticateUser(tt.username, tt.password)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("authenticateUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOIDCServer_hashPassword(t *testing.T) {
	server := createTestOIDCServer()

	tests := []struct {
		name     string
		password string
	}{
		{name: "simple password", password: "password123"},
		{name: "complex password", password: "P@ssw0rd!@#$%^&*()"},
		{name: "empty password", password: ""},
		{name: "unicode password", password: "пароль123🔑"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashed, err := server.hashPassword(tt.password)
			if err != nil {
				t.Errorf("hashPassword() error = %v", err)
				return
			}

			// Verify the hash can be used to validate the original password
			err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(tt.password))
			if err != nil {
				t.Errorf("Generated hash does not validate original password: %v", err)
			}

			// Verify the hash is different from the password
			if hashed == tt.password {
				t.Error("Hash should be different from the original password")
			}

			// Verify bcrypt hash format
			if len(hashed) < 59 { // bcrypt hashes are typically 60 characters
				t.Errorf("Hash seems too short: %d characters", len(hashed))
			}
		})
	}
}

func TestOIDCServer_getUserClaims(t *testing.T) {
	server := createTestOIDCServer()
	alice := &server.config.Users[0]
	charlie := &server.config.Users[2] // Has more claims

	tests := []struct {
		name   string
		user   *User
		scopes []string
		want   map[string]interface{}
	}{
		{
			name:   "openid scope only",
			user:   alice,
			scopes: []string{"openid"},
			want: map[string]interface{}{
				"sub": "alice",
			},
		},
		{
			name:   "openid and profile scopes",
			user:   alice,
			scopes: []string{"openid", "profile"},
			want: map[string]interface{}{
				"sub":         "alice",
				"given_name":  "Alice",
				"family_name": "Smith",
				"name":        "Alice Smith",
			},
		},
		{
			name:   "openid and email scopes",
			user:   alice,
			scopes: []string{"openid", "email"},
			want: map[string]interface{}{
				"sub":   "alice",
				"email": "alice@example.com",
			},
		},
		{
			name:   "all standard scopes",
			user:   alice,
			scopes: []string{"openid", "profile", "email"},
			want: map[string]interface{}{
				"sub":         "alice",
				"email":       "alice@example.com",
				"given_name":  "Alice",
				"family_name": "Smith",
				"name":        "Alice Smith",
			},
		},
		{
			name:   "user with extended claims - address and phone",
			user:   charlie,
			scopes: []string{"openid", "profile", "email", "address", "phone"},
			want: map[string]interface{}{
				"sub":                   "charlie",
				"email":                 "charlie@example.com",
				"email_verified":        true, // Should be converted to boolean
				"phone_number":          "+1234567890",
				"phone_number_verified": false, // Should be converted to boolean
				"address":               "123 Main St",
				"nickname":              "Chuck",
				"preferred_username":    "charles",
			},
		},
		{
			name:   "unknown scope ignored",
			user:   alice,
			scopes: []string{"openid", "unknown_scope"},
			want: map[string]interface{}{
				"sub": "alice",
			},
		},
		{
			name:   "empty scopes",
			user:   alice,
			scopes: []string{},
			want: map[string]interface{}{
				"sub": "alice",
			},
		},
		{
			name:   "user without explicit sub claim",
			user:   &User{Username: "testuser", Claims: map[string]string{"email": "test@example.com"}},
			scopes: []string{"openid", "email"},
			want: map[string]interface{}{
				"sub":   "testuser", // Should default to username
				"email": "test@example.com",
			},
		},
		{
			name:   "user with nil claims",
			user:   &User{Username: "testuser", Claims: nil},
			scopes: []string{"openid"},
			want: map[string]interface{}{
				"sub": "testuser", // Should default to username
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := server.getUserClaims(tt.user, tt.scopes)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUserClaims() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserClaims_ScopeHandling(t *testing.T) {
	server := createTestOIDCServer()
	user := &User{
		Username: "testuser",
		Claims: map[string]string{
			"sub":                   "testuser",
			"name":                  "Test User",
			"given_name":            "Test",
			"family_name":           "User",
			"nickname":              "Testy",
			"preferred_username":    "test",
			"email":                 "test@example.com",
			"email_verified":        "true",
			"phone_number":          "+1234567890",
			"phone_number_verified": "false",
			"address":               "123 Test St",
		},
	}

	t.Run("profile scope includes all profile claims", func(t *testing.T) {
		claims := server.getUserClaims(user, []string{"openid", "profile"})

		expectedProfileClaims := []string{"name", "given_name", "family_name", "nickname", "preferred_username"}
		for _, claim := range expectedProfileClaims {
			if _, exists := claims[claim]; !exists {
				t.Errorf("Profile scope should include claim '%s'", claim)
			}
		}

		// Should not include email claims
		if _, exists := claims["email"]; exists {
			t.Error("Profile scope should not include email claim")
		}
	})

	t.Run("email scope includes email claims", func(t *testing.T) {
		claims := server.getUserClaims(user, []string{"openid", "email"})

		if claims["email"] != "test@example.com" {
			t.Error("Email scope should include email claim")
		}

		if claims["email_verified"] != true {
			t.Error("Email scope should include email_verified as boolean true")
		}

		// Should not include profile claims
		if _, exists := claims["name"]; exists {
			t.Error("Email scope should not include profile claims")
		}
	})

	t.Run("phone scope includes phone claims", func(t *testing.T) {
		claims := server.getUserClaims(user, []string{"openid", "phone"})

		if claims["phone_number"] != "+1234567890" {
			t.Error("Phone scope should include phone_number claim")
		}

		if claims["phone_number_verified"] != false {
			t.Error("Phone scope should include phone_number_verified as boolean false")
		}
	})

	t.Run("address scope includes address claim", func(t *testing.T) {
		claims := server.getUserClaims(user, []string{"openid", "address"})

		if claims["address"] != "123 Test St" {
			t.Error("Address scope should include address claim")
		}
	})
}

func TestGetUserClaims_BooleanConversion(t *testing.T) {
	server := createTestOIDCServer()

	tests := []struct {
		name           string
		claimValue     string
		expectedResult interface{}
	}{
		{name: "true string", claimValue: "true", expectedResult: true},
		{name: "false string", claimValue: "false", expectedResult: false},
		{name: "True with capital", claimValue: "True", expectedResult: false}, // Only "true" converts to true
		{name: "empty string", claimValue: "", expectedResult: false},          // Empty string converts to false
		{name: "other value", claimValue: "yes", expectedResult: false},        // Any other value converts to false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				Username: "test",
				Claims: map[string]string{
					"sub":            "test",
					"email_verified": tt.claimValue,
				},
			}

			claims := server.getUserClaims(user, []string{"openid", "email"})

			if claims["email_verified"] != tt.expectedResult {
				t.Errorf("Expected email_verified to be %v (%T), got %v (%T)",
					tt.expectedResult, tt.expectedResult, claims["email_verified"], claims["email_verified"])
			}
		})
	}
}
