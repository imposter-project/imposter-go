package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	PKCEMethodPlain = "plain"
	PKCEMethodS256  = "S256"
)

func validatePKCE(codeChallenge, codeChallengeMethod, codeVerifier string) error {
	if codeChallenge == "" {
		// PKCE is optional, so no challenge means no verification needed
		return nil
	}

	if codeVerifier == "" {
		return fmt.Errorf("code_verifier is required when code_challenge is present")
	}

	// Validate code_verifier format (RFC 7636)
	if len(codeVerifier) < 43 || len(codeVerifier) > 128 {
		return fmt.Errorf("code_verifier must be between 43 and 128 characters")
	}

	// Check if code_verifier contains only valid characters
	for _, char := range codeVerifier {
		if !isValidPKCEChar(char) {
			return fmt.Errorf("code_verifier contains invalid characters")
		}
	}

	switch codeChallengeMethod {
	case PKCEMethodPlain, "":
		// Plain method: challenge must equal verifier
		if codeChallenge != codeVerifier {
			return fmt.Errorf("code_challenge does not match code_verifier")
		}
	case PKCEMethodS256:
		// S256 method: challenge must be base64url(sha256(verifier))
		hash := sha256.Sum256([]byte(codeVerifier))
		expectedChallenge := base64URLEncode(hash[:])
		if codeChallenge != expectedChallenge {
			return fmt.Errorf("code_challenge does not match SHA256 of code_verifier")
		}
	default:
		return fmt.Errorf("unsupported code_challenge_method: %s", codeChallengeMethod)
	}

	return nil
}

func isValidPKCEChar(char rune) bool {
	// RFC 7636: code_verifier should use unreserved characters [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
	return (char >= 'A' && char <= 'Z') ||
		(char >= 'a' && char <= 'z') ||
		(char >= '0' && char <= '9') ||
		char == '-' || char == '.' || char == '_' || char == '~'
}

func base64URLEncode(data []byte) string {
	// Base64url encoding: use URL-safe alphabet and remove padding
	encoded := base64.URLEncoding.EncodeToString(data)
	return strings.TrimRight(encoded, "=")
}

func validateCodeChallenge(codeChallenge, codeChallengeMethod string) error {
	if codeChallenge == "" {
		return nil // PKCE is optional
	}

	// Validate code_challenge format
	if len(codeChallenge) < 43 || len(codeChallenge) > 128 {
		return fmt.Errorf("code_challenge must be between 43 and 128 characters")
	}

	// Validate code_challenge_method
	if codeChallengeMethod != "" && codeChallengeMethod != PKCEMethodPlain && codeChallengeMethod != PKCEMethodS256 {
		return fmt.Errorf("unsupported code_challenge_method: %s", codeChallengeMethod)
	}

	return nil
}
