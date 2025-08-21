package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadOIDCConfig(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "oidc-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name        string
		setupFiles  func() error
		expectError bool
		validate    func(*OIDCConfig) error
	}{
		{
			name: "valid config file (oidc-users.yaml)",
			setupFiles: func() error {
				content := `
users:
  - username: "testuser"
    password: "testpass"
    claims:
      sub: "testuser"
      email: "test@example.com"
clients:
  - client_id: "testclient"
    client_secret: "testsecret"
    redirect_uris:
      - "http://localhost:8080/callback"
`
				return os.WriteFile(filepath.Join(tmpDir, "oidc-users.yaml"), []byte(content), 0644)
			},
			expectError: false,
			validate: func(config *OIDCConfig) error {
				if len(config.Users) != 1 || config.Users[0].Username != "testuser" {
					t.Errorf("Expected 1 user with username 'testuser', got %+v", config.Users)
				}
				if len(config.Clients) != 1 || config.Clients[0].ClientID != "testclient" {
					t.Errorf("Expected 1 client with ID 'testclient', got %+v", config.Clients)
				}
				return nil
			},
		},
		{
			name: "alternative config file (oidc.yaml)",
			setupFiles: func() error {
				content := `
users:
  - username: "altuser"
    password: "altpass"
    claims:
      sub: "altuser"
clients:
  - client_id: "altclient"
    redirect_uris:
      - "http://localhost:3000/callback"
`
				return os.WriteFile(filepath.Join(tmpDir, "oidc.yaml"), []byte(content), 0644)
			},
			expectError: false,
			validate: func(config *OIDCConfig) error {
				if len(config.Users) != 1 || config.Users[0].Username != "altuser" {
					t.Errorf("Expected 1 user with username 'altuser', got %+v", config.Users)
				}
				return nil
			},
		},
		{
			name: "no config file - uses default",
			setupFiles: func() error {
				return nil // No files created
			},
			expectError: false,
			validate: func(config *OIDCConfig) error {
				if len(config.Users) < 2 {
					t.Errorf("Expected default config with multiple users, got %d users", len(config.Users))
				}
				if config.Users[0].Username != "alice" && config.Users[1].Username != "alice" {
					t.Errorf("Expected default config to contain 'alice' user")
				}
				return nil
			},
		},
		{
			name: "invalid YAML format",
			setupFiles: func() error {
				content := `invalid: yaml: content: [}`
				return os.WriteFile(filepath.Join(tmpDir, "oidc-users.yaml"), []byte(content), 0644)
			},
			expectError: true,
		},
		{
			name: "missing required fields",
			setupFiles: func() error {
				content := `
users:
  - username: ""  # Invalid empty username
    password: "test"
clients:
  - client_id: "test"
    redirect_uris: []  # Invalid empty redirect_uris
`
				return os.WriteFile(filepath.Join(tmpDir, "oidc-users.yaml"), []byte(content), 0644)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean directory
			files, _ := os.ReadDir(tmpDir)
			for _, file := range files {
				os.Remove(filepath.Join(tmpDir, file.Name()))
			}

			// Setup test files
			if err := tt.setupFiles(); err != nil {
				t.Fatal(err)
			}

			// Test the function
			config, err := loadOIDCConfig(tmpDir)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && config != nil && tt.validate != nil {
				if err := tt.validate(config); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *OIDCConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: &OIDCConfig{
				Users: []User{
					{Username: "user1", Password: "pass1", Claims: map[string]string{"sub": "user1"}},
				},
				Clients: []Client{
					{ClientID: "client1", RedirectURIs: []string{"http://localhost:8080/callback"}},
				},
			},
			expectError: false,
		},
		{
			name: "no users",
			config: &OIDCConfig{
				Users: []User{},
				Clients: []Client{
					{ClientID: "client1", RedirectURIs: []string{"http://localhost:8080/callback"}},
				},
			},
			expectError: true,
		},
		{
			name: "no clients",
			config: &OIDCConfig{
				Users: []User{
					{Username: "user1", Password: "pass1"},
				},
				Clients: []Client{},
			},
			expectError: true,
		},
		{
			name: "empty username",
			config: &OIDCConfig{
				Users: []User{
					{Username: "", Password: "pass1"},
				},
				Clients: []Client{
					{ClientID: "client1", RedirectURIs: []string{"http://localhost:8080/callback"}},
				},
			},
			expectError: true,
		},
		{
			name: "empty password",
			config: &OIDCConfig{
				Users: []User{
					{Username: "user1", Password: ""},
				},
				Clients: []Client{
					{ClientID: "client1", RedirectURIs: []string{"http://localhost:8080/callback"}},
				},
			},
			expectError: true,
		},
		{
			name: "empty client ID",
			config: &OIDCConfig{
				Users: []User{
					{Username: "user1", Password: "pass1"},
				},
				Clients: []Client{
					{ClientID: "", RedirectURIs: []string{"http://localhost:8080/callback"}},
				},
			},
			expectError: true,
		},
		{
			name: "no redirect URIs",
			config: &OIDCConfig{
				Users: []User{
					{Username: "user1", Password: "pass1"},
				},
				Clients: []Client{
					{ClientID: "client1", RedirectURIs: []string{}},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := getDefaultConfig()

	// Test that default config is valid
	if err := validateConfig(config); err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}

	// Test specific expectations
	if len(config.Users) < 2 {
		t.Errorf("Expected at least 2 default users, got %d", len(config.Users))
	}

	if len(config.Clients) < 1 {
		t.Errorf("Expected at least 1 default client, got %d", len(config.Clients))
	}

	// Check that default users have required claims
	for i, user := range config.Users {
		if user.Claims == nil {
			t.Errorf("User at index %d should have claims", i)
		}
		if _, hasSub := user.Claims["sub"]; !hasSub {
			t.Errorf("User %s should have 'sub' claim", user.Username)
		}
	}
}

func TestOIDCConfig_GetUser(t *testing.T) {
	config := &OIDCConfig{
		Users: []User{
			{Username: "alice", Password: "pass1"},
			{Username: "bob", Password: "pass2"},
		},
	}

	tests := []struct {
		name     string
		username string
		want     *User
	}{
		{
			name:     "existing user alice",
			username: "alice",
			want:     &User{Username: "alice", Password: "pass1"},
		},
		{
			name:     "existing user bob",
			username: "bob",
			want:     &User{Username: "bob", Password: "pass2"},
		},
		{
			name:     "non-existing user",
			username: "charlie",
			want:     nil,
		},
		{
			name:     "empty username",
			username: "",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.GetUser(tt.username)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOIDCConfig_GetClient(t *testing.T) {
	config := &OIDCConfig{
		Clients: []Client{
			{ClientID: "client1", ClientSecret: "secret1"},
			{ClientID: "client2", ClientSecret: "secret2"},
		},
	}

	tests := []struct {
		name     string
		clientID string
		want     *Client
	}{
		{
			name:     "existing client1",
			clientID: "client1",
			want:     &Client{ClientID: "client1", ClientSecret: "secret1"},
		},
		{
			name:     "existing client2",
			clientID: "client2",
			want:     &Client{ClientID: "client2", ClientSecret: "secret2"},
		},
		{
			name:     "non-existing client",
			clientID: "client3",
			want:     nil,
		},
		{
			name:     "empty client ID",
			clientID: "",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.GetClient(tt.clientID)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_IsValidRedirectURI(t *testing.T) {
	client := &Client{
		ClientID: "test",
		RedirectURIs: []string{
			"http://localhost:8080/callback",
			"https://example.com/oauth/callback",
			"com.example.app://oauth",
		},
	}

	tests := []struct {
		name string
		uri  string
		want bool
	}{
		{
			name: "valid http URI",
			uri:  "http://localhost:8080/callback",
			want: true,
		},
		{
			name: "valid https URI",
			uri:  "https://example.com/oauth/callback",
			want: true,
		},
		{
			name: "valid custom scheme URI",
			uri:  "com.example.app://oauth",
			want: true,
		},
		{
			name: "invalid URI",
			uri:  "http://malicious.com/callback",
			want: false,
		},
		{
			name: "empty URI",
			uri:  "",
			want: false,
		},
		{
			name: "similar but different URI",
			uri:  "http://localhost:8080/callback/extra",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.IsValidRedirectURI(tt.uri)
			if got != tt.want {
				t.Errorf("IsValidRedirectURI() = %v, want %v", got, tt.want)
			}
		})
	}
}
