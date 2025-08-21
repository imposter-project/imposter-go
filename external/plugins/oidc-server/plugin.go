package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
)

type OIDCServer struct {
	logger    hclog.Logger
	config    *OIDCConfig
	serverURL string
	sessions  map[string]*AuthSession
	codes     map[string]*AuthCode
	tokens    map[string]*AccessToken
	mutex     sync.RWMutex
	jwtSecret []byte
}

type AuthSession struct {
	ID              string
	ClientID        string
	RedirectURI     string
	Scope           string
	State           string
	Nonce           string
	CodeChallenge   string
	ChallengeMethod string
	CreatedAt       time.Time
	ExpiresAt       time.Time
}

type AuthCode struct {
	Code            string
	ClientID        string
	RedirectURI     string
	UserID          string
	Scope           string
	Nonce           string
	CodeChallenge   string
	ChallengeMethod string
	CreatedAt       time.Time
	ExpiresAt       time.Time
}

type AccessToken struct {
	Token     string
	UserID    string
	ClientID  string
	Scope     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (o *OIDCServer) Configure(cfg shared.ExternalConfig) error {
	o.logger.Trace("configuring OIDC server plugin")

	// Store server URL from configuration
	o.serverURL = cfg.Server.URL
	if o.serverURL == "" {
		o.serverURL = "http://localhost:8080" // fallback
	}
	o.logger.Debug("using server URL", "url", o.serverURL)

	o.sessions = make(map[string]*AuthSession)
	o.codes = make(map[string]*AuthCode)
	o.tokens = make(map[string]*AccessToken)

	// Generate JWT signing key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate JWT signing key: %w", err)
	}
	o.jwtSecret = key

	// Load OIDC configuration from the first config directory
	if len(cfg.Configs) > 0 {
		configDir := cfg.Configs[0].ConfigDir
		o.logger.Debug("loading OIDC config from directory", "configDir", configDir)

		config, err := loadOIDCConfig(configDir)
		if err != nil {
			return fmt.Errorf("failed to load OIDC config: %w", err)
		}
		o.config = config

		o.logger.Info("OIDC server configured", "users", len(config.Users), "clients", len(config.Clients))
	} else {
		// Use default configuration if none provided
		o.config = getDefaultConfig()
		o.logger.Warn("using default OIDC configuration")
	}

	return nil
}

func (o *OIDCServer) Handle(args shared.HandlerRequest) shared.HandlerResponse {
	o.logger.Debug("handling request", "method", args.Method, "path", args.Path)

	// Parse the path to determine the endpoint
	switch args.Path {
	case "/oidc/authorize":
		return o.handleAuthorize(args)
	case "/oidc/token":
		return o.handleToken(args)
	case "/oidc/userinfo":
		return o.handleUserInfo(args)
	case "/.well-known/openid-configuration":
		return o.handleDiscovery(args)
	default:
		return shared.HandlerResponse{StatusCode: 404, Body: []byte("Not Found")}
	}
}

func (o *OIDCServer) handleDiscovery(args shared.HandlerRequest) shared.HandlerResponse {
	if !strings.EqualFold(args.Method, "GET") {
		return shared.HandlerResponse{StatusCode: 405, Body: []byte("Method Not Allowed")}
	}

	// Build endpoints using configured server URL
	issuer := o.serverURL
	authzEndpoint := o.serverURL + "/oidc/authorize"
	tokenEndpoint := o.serverURL + "/oidc/token"
	userinfoEndpoint := o.serverURL + "/oidc/userinfo"

	body := fmt.Sprintf(`{
		"issuer": "%s",
		"authorization_endpoint": "%s", 
		"token_endpoint": "%s",
		"userinfo_endpoint": "%s",
		"response_types_supported": ["code"],
		"subject_types_supported": ["public"],
		"id_token_signing_alg_values_supported": ["HS256"],
		"scopes_supported": ["openid", "profile", "email"],
		"claims_supported": ["sub", "name", "given_name", "family_name", "email"],
		"code_challenge_methods_supported": ["S256", "plain"]
	}`, issuer, authzEndpoint, tokenEndpoint, userinfoEndpoint)

	return shared.HandlerResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(body),
	}
}

func (o *OIDCServer) generateSessionID() string {
	return uuid.New().String()
}

func (o *OIDCServer) generateAuthCode() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (o *OIDCServer) generateAccessToken() string {
	return uuid.New().String()
}

func (o *OIDCServer) cleanupExpired() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	now := time.Now()

	// Clean up expired sessions
	for id, session := range o.sessions {
		if now.After(session.ExpiresAt) {
			delete(o.sessions, id)
		}
	}

	// Clean up expired codes
	for code, authCode := range o.codes {
		if now.After(authCode.ExpiresAt) {
			delete(o.codes, code)
		}
	}

	// Clean up expired tokens
	for token, accessToken := range o.tokens {
		if now.After(accessToken.ExpiresAt) {
			delete(o.tokens, token)
		}
	}
}

func parseFormData(body []byte) (url.Values, error) {
	return url.ParseQuery(string(body))
}
