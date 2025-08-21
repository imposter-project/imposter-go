package main

import (
	"crypto/rand"
	"crypto/sha256"
	"strings"
	"testing"
)

func TestValidatePKCE(t *testing.T) {
	// Helper to generate valid code verifier
	generateCodeVerifier := func() string {
		bytes := make([]byte, 32)
		rand.Read(bytes)
		return base64URLEncode(bytes)
	}

	// Helper to generate S256 challenge from verifier
	generateS256Challenge := func(verifier string) string {
		hash := sha256.Sum256([]byte(verifier))
		return base64URLEncode(hash[:])
	}

	tests := []struct {
		name                string
		codeChallenge       string
		codeChallengeMethod string
		codeVerifier        string
		expectError         bool
	}{
		{
			name:                "no PKCE - valid",
			codeChallenge:       "",
			codeChallengeMethod: "",
			codeVerifier:        "",
			expectError:         false,
		},
		{
			name:                "plain method - valid",
			codeChallenge:       "test-challenge-12345678901234567890123456789012345",
			codeChallengeMethod: "plain",
			codeVerifier:        "test-challenge-12345678901234567890123456789012345",
			expectError:         false,
		},
		{
			name:                "plain method - empty method defaults to plain",
			codeChallenge:       "test-challenge-12345678901234567890123456789012345",
			codeChallengeMethod: "",
			codeVerifier:        "test-challenge-12345678901234567890123456789012345",
			expectError:         false,
		},
		{
			name: "S256 method - valid",
			codeChallenge: func() string {
				verifier := generateCodeVerifier()
				return generateS256Challenge(verifier)
			}(),
			codeChallengeMethod: "S256",
			codeVerifier:        generateCodeVerifier(),
			expectError:         false,
		},
		{
			name:                "challenge present but no verifier",
			codeChallenge:       "test-challenge-12345678901234567890123456789012345",
			codeChallengeMethod: "plain",
			codeVerifier:        "",
			expectError:         true,
		},
		{
			name:                "verifier too short",
			codeChallenge:       "short",
			codeChallengeMethod: "plain",
			codeVerifier:        "short",
			expectError:         true,
		},
		{
			name:                "verifier too long",
			codeChallenge:       strings.Repeat("x", 129),
			codeChallengeMethod: "plain",
			codeVerifier:        strings.Repeat("x", 129),
			expectError:         true,
		},
		{
			name:                "verifier invalid characters",
			codeChallenge:       "test@#$%^&*()+={}[]|\\:;\"'<>?,./",
			codeChallengeMethod: "plain",
			codeVerifier:        "test@#$%^&*()+={}[]|\\:;\"'<>?,./",
			expectError:         true,
		},
		{
			name:                "plain method mismatch",
			codeChallenge:       "challenge",
			codeChallengeMethod: "plain",
			codeVerifier:        "different-verifier",
			expectError:         true,
		},
		{
			name:                "S256 method mismatch",
			codeChallenge:       "invalid-hash",
			codeChallengeMethod: "S256",
			codeVerifier:        generateCodeVerifier(),
			expectError:         true,
		},
		{
			name:                "unsupported method",
			codeChallenge:       "test-challenge-12345678901234567890123456789012345",
			codeChallengeMethod: "MD5",
			codeVerifier:        "test-challenge-12345678901234567890123456789012345",
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For S256 tests, we need to generate matching challenge/verifier pairs
			if tt.codeChallengeMethod == "S256" && tt.name == "S256 method - valid" {
				verifier := generateCodeVerifier()
				challenge := generateS256Challenge(verifier)
				err := validatePKCE(challenge, tt.codeChallengeMethod, verifier)
				if tt.expectError && err == nil {
					t.Error("Expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				return
			}

			err := validatePKCE(tt.codeChallenge, tt.codeChallengeMethod, tt.codeVerifier)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateCodeChallenge(t *testing.T) {
	tests := []struct {
		name                string
		codeChallenge       string
		codeChallengeMethod string
		expectError         bool
	}{
		{
			name:                "no PKCE",
			codeChallenge:       "",
			codeChallengeMethod: "",
			expectError:         false,
		},
		{
			name:                "valid challenge with plain method",
			codeChallenge:       "test-challenge-12345678901234567890123456789012345",
			codeChallengeMethod: "plain",
			expectError:         false,
		},
		{
			name:                "valid challenge with S256 method",
			codeChallenge:       "test-challenge-12345678901234567890123456789012345",
			codeChallengeMethod: "S256",
			expectError:         false,
		},
		{
			name:                "valid challenge with empty method",
			codeChallenge:       "test-challenge-12345678901234567890123456789012345",
			codeChallengeMethod: "",
			expectError:         false,
		},
		{
			name:                "challenge too short",
			codeChallenge:       "short",
			codeChallengeMethod: "plain",
			expectError:         true,
		},
		{
			name:                "challenge too long",
			codeChallenge:       strings.Repeat("x", 129),
			codeChallengeMethod: "plain",
			expectError:         true,
		},
		{
			name:                "unsupported method",
			codeChallenge:       "test-challenge-12345678901234567890123456789012345",
			codeChallengeMethod: "MD5",
			expectError:         true,
		},
		{
			name:                "unsupported method - SHA1",
			codeChallenge:       "test-challenge-12345678901234567890123456789012345",
			codeChallengeMethod: "SHA1",
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCodeChallenge(tt.codeChallenge, tt.codeChallengeMethod)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestIsValidPKCEChar(t *testing.T) {
	tests := []struct {
		name string
		char rune
		want bool
	}{
		// Valid characters
		{name: "uppercase A", char: 'A', want: true},
		{name: "uppercase Z", char: 'Z', want: true},
		{name: "lowercase a", char: 'a', want: true},
		{name: "lowercase z", char: 'z', want: true},
		{name: "digit 0", char: '0', want: true},
		{name: "digit 9", char: '9', want: true},
		{name: "hyphen", char: '-', want: true},
		{name: "dot", char: '.', want: true},
		{name: "underscore", char: '_', want: true},
		{name: "tilde", char: '~', want: true},

		// Invalid characters
		{name: "space", char: ' ', want: false},
		{name: "plus", char: '+', want: false},
		{name: "forward slash", char: '/', want: false},
		{name: "equals", char: '=', want: false},
		{name: "at sign", char: '@', want: false},
		{name: "hash", char: '#', want: false},
		{name: "dollar", char: '$', want: false},
		{name: "percent", char: '%', want: false},
		{name: "caret", char: '^', want: false},
		{name: "ampersand", char: '&', want: false},
		{name: "asterisk", char: '*', want: false},
		{name: "left paren", char: '(', want: false},
		{name: "right paren", char: ')', want: false},
		{name: "left bracket", char: '[', want: false},
		{name: "right bracket", char: ']', want: false},
		{name: "left brace", char: '{', want: false},
		{name: "right brace", char: '}', want: false},
		{name: "pipe", char: '|', want: false},
		{name: "backslash", char: '\\', want: false},
		{name: "colon", char: ':', want: false},
		{name: "semicolon", char: ';', want: false},
		{name: "quote", char: '"', want: false},
		{name: "single quote", char: '\'', want: false},
		{name: "less than", char: '<', want: false},
		{name: "greater than", char: '>', want: false},
		{name: "comma", char: ',', want: false},
		{name: "question mark", char: '?', want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidPKCEChar(tt.char)
			if got != tt.want {
				t.Errorf("isValidPKCEChar(%c) = %v, want %v", tt.char, got, tt.want)
			}
		})
	}
}

func TestBase64URLEncode(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty input",
			input: []byte{},
			want:  "",
		},
		{
			name:  "simple text",
			input: []byte("hello"),
			want:  "aGVsbG8",
		},
		{
			name:  "text requiring padding removal",
			input: []byte("hello world"),
			want:  "aGVsbG8gd29ybGQ",
		},
		{
			name:  "binary data",
			input: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE},
			want:  "AAECA__-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := base64URLEncode(tt.input)
			if got != tt.want {
				t.Errorf("base64URLEncode() = %v, want %v", got, tt.want)
			}

			// Verify it's valid base64url (no padding, URL-safe characters)
			if strings.ContainsAny(got, "=+/") {
				t.Errorf("base64URLEncode() result contains invalid characters: %v", got)
			}
		})
	}
}

func TestPKCEIntegration(t *testing.T) {
	// Test full PKCE flow integration
	t.Run("complete PKCE S256 flow", func(t *testing.T) {
		// Generate a valid code verifier
		verifierBytes := make([]byte, 32)
		rand.Read(verifierBytes)
		verifier := base64URLEncode(verifierBytes)

		// Generate the S256 challenge
		hash := sha256.Sum256([]byte(verifier))
		challenge := base64URLEncode(hash[:])

		// Validate the challenge during authorization
		err := validateCodeChallenge(challenge, "S256")
		if err != nil {
			t.Fatalf("Challenge validation failed: %v", err)
		}

		// Validate PKCE during token exchange
		err = validatePKCE(challenge, "S256", verifier)
		if err != nil {
			t.Fatalf("PKCE validation failed: %v", err)
		}
	})

	t.Run("complete PKCE plain flow", func(t *testing.T) {
		// Use the verifier as the challenge for plain method
		verifierBytes := make([]byte, 32)
		rand.Read(verifierBytes)
		verifier := base64URLEncode(verifierBytes)
		challenge := verifier

		// Validate the challenge during authorization
		err := validateCodeChallenge(challenge, "plain")
		if err != nil {
			t.Fatalf("Challenge validation failed: %v", err)
		}

		// Validate PKCE during token exchange
		err = validatePKCE(challenge, "plain", verifier)
		if err != nil {
			t.Fatalf("PKCE validation failed: %v", err)
		}
	})

	t.Run("PKCE attack prevention", func(t *testing.T) {
		// Generate legitimate PKCE pair
		verifierBytes := make([]byte, 32)
		rand.Read(verifierBytes)
		legitimateVerifier := base64URLEncode(verifierBytes)
		hash := sha256.Sum256([]byte(legitimateVerifier))
		challenge := base64URLEncode(hash[:])

		// Try to use wrong verifier (attack scenario)
		attackerVerifierBytes := make([]byte, 32)
		rand.Read(attackerVerifierBytes)
		attackerVerifier := base64URLEncode(attackerVerifierBytes)

		// This should fail
		err := validatePKCE(challenge, "S256", attackerVerifier)
		if err == nil {
			t.Fatal("PKCE validation should have failed with wrong verifier")
		}
	})
}
