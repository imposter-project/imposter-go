package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
)

type OIDCServer struct {
	logger          hclog.Logger
	config          *OIDCConfig
	serverURL       string
	sessions        map[string]*AuthSession
	codes           map[string]*AuthCode
	tokens          map[string]*AccessToken
	mutex           sync.RWMutex
	jwtSecret       []byte
	privateKey      *rsa.PrivateKey
	publicKey       *rsa.PublicKey
	cachedJWKS      []byte
	cachedDiscovery []byte
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

	// Generate JWT signing key for HS256 (will be used if config defaults to HS256)
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate JWT signing key: %w", err)
	}
	o.jwtSecret = key

	if len(cfg.Configs) > 0 {
		firstConfig := cfg.Configs[0]

		// Try to load from plugin config block first
		if len(firstConfig.PluginConfig) > 0 {
			o.logger.Debug("loading OIDC config from plugin config block")
			config, err := loadOIDCConfig(firstConfig.PluginConfig)
			if err != nil {
				return fmt.Errorf("failed to load OIDC config from plugin config block: %w", err)
			}
			o.config = config
			o.logger.Info("OIDC server configured from plugin config block", "users", len(config.Users), "clients", len(config.Clients))
		} else {
			o.logger.Warn("no OIDC config found in plugin config block")
		}
	}

	if o.config == nil {
		o.logger.Warn("no OIDC configuration provided, using default configuration")
		o.config = getDefaultConfig()
	}

	// Setup JWT signing keys based on algorithm
	if err := o.setupJWTKeys(); err != nil {
		return fmt.Errorf("failed to setup JWT keys: %w", err)
	}

	// Cache the discovery document
	if err := o.CacheDiscoveryDocument(); err != nil {
		return fmt.Errorf("failed to cache discovery document: %w", err)
	}

	return nil
}

func (o *OIDCServer) setupJWTKeys() error {
	jwtConfig := o.config.JWTConfig
	if jwtConfig == nil {
		return fmt.Errorf("JWT configuration is nil")
	}

	switch jwtConfig.Algorithm {
	case "HS256":
		// jwtSecret is already generated, nothing more to do
		o.logger.Debug("using HS256 algorithm for JWT signing")
	case "RS256":
		o.logger.Debug("using RS256 algorithm for JWT signing")

		// Parse private key
		privateKeyBlock, _ := pem.Decode([]byte(jwtConfig.PrivateKeyPEM))
		if privateKeyBlock == nil {
			return fmt.Errorf("failed to decode private key PEM")
		}

		privateKey, err := x509.ParsePKCS8PrivateKey(privateKeyBlock.Bytes)
		if err != nil {
			// Try PKCS1 format as fallback
			privateKey, err = x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
			if err != nil {
				return fmt.Errorf("failed to parse private key: %w", err)
			}
		}

		rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("private key is not an RSA key")
		}
		o.privateKey = rsaPrivateKey

		// Parse public key
		publicKeyBlock, _ := pem.Decode([]byte(jwtConfig.PublicKeyPEM))
		if publicKeyBlock == nil {
			return fmt.Errorf("failed to decode public key PEM")
		}

		publicKeyInterface, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse public key: %w", err)
		}

		rsaPublicKey, ok := publicKeyInterface.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("public key is not an RSA key")
		}
		o.publicKey = rsaPublicKey

		// Pre-generate and cache JWKS
		if err := o.cacheJWKS(); err != nil {
			return fmt.Errorf("failed to cache JWKS: %w", err)
		}

		o.logger.Info("successfully loaded RSA key pair for JWT signing", "key_id", jwtConfig.KeyID)
	default:
		return fmt.Errorf("unsupported JWT algorithm: %s", jwtConfig.Algorithm)
	}

	return nil
}

func (o *OIDCServer) cacheJWKS() error {
	jwk, err := o.generateJWK()
	if err != nil {
		return fmt.Errorf("failed to generate JWK: %w", err)
	}

	jwks := fmt.Sprintf(`{"keys": [%s]}`, jwk)
	o.cachedJWKS = []byte(jwks)
	o.logger.Debug("JWKS cached successfully")
	return nil
}

func (o *OIDCServer) CacheDiscoveryDocument() error {
	// Build discovery document as a map
	discovery := map[string]interface{}{
		"issuer":                           o.serverURL,
		"authorization_endpoint":           o.serverURL + "/oidc/authorize",
		"token_endpoint":                   o.serverURL + "/oidc/token",
		"userinfo_endpoint":                o.serverURL + "/oidc/userinfo",
		"jwks_uri":                         o.serverURL + "/oidc/jwks",
		"response_types_supported":         []string{"code"},
		"subject_types_supported":          []string{"public"},
		"scopes_supported":                 []string{"openid", "profile", "email"},
		"claims_supported":                 []string{"sub", "name", "given_name", "family_name", "email"},
		"code_challenge_methods_supported": []string{"S256", "plain"},
	}

	// Set signing algorithms based on configuration
	if o.config.JWTConfig != nil && o.config.JWTConfig.Algorithm == "RS256" {
		discovery["id_token_signing_alg_values_supported"] = []string{"RS256"}
	} else {
		discovery["id_token_signing_alg_values_supported"] = []string{"HS256"}
	}

	// Marshal to JSON
	discoveryJSON, err := json.Marshal(discovery)
	if err != nil {
		return fmt.Errorf("failed to marshal discovery document: %w", err)
	}

	o.cachedDiscovery = discoveryJSON
	o.logger.Debug("discovery document cached successfully")
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
	case "/oidc/jwks":
		return o.handleJWKS(args)
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

	// Return cached discovery document (should be populated by Configure or test setup)
	return shared.HandlerResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: o.cachedDiscovery,
	}
}

func (o *OIDCServer) handleJWKS(args shared.HandlerRequest) shared.HandlerResponse {
	if !strings.EqualFold(args.Method, "GET") {
		return shared.HandlerResponse{StatusCode: 405, Body: []byte("Method Not Allowed")}
	}

	// Only provide JWKS for RS256
	if o.config.JWTConfig == nil || o.config.JWTConfig.Algorithm != "RS256" {
		return shared.HandlerResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: []byte(`{"keys": []}`),
		}
	}

	// Return cached JWKS
	return shared.HandlerResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: o.cachedJWKS,
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

func (o *OIDCServer) generateJWK() (string, error) {
	if o.publicKey == nil {
		return "", fmt.Errorf("no RSA public key available")
	}

	// Convert RSA public key components to base64url encoding
	nBytes := o.publicKey.N.Bytes()
	eBytes := big.NewInt(int64(o.publicKey.E)).Bytes()

	n := base64.RawURLEncoding.EncodeToString(nBytes)
	e := base64.RawURLEncoding.EncodeToString(eBytes)

	jwk := map[string]interface{}{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": o.config.JWTConfig.KeyID,
		"n":   n,
		"e":   e,
	}

	jwkBytes, err := json.Marshal(jwk)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWK: %w", err)
	}

	return string(jwkBytes), nil
}

func parseFormData(body []byte) (url.Values, error) {
	return url.ParseQuery(string(body))
}
