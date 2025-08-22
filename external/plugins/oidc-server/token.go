package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/imposter-project/imposter-go/external/shared"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type TokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func (o *OIDCServer) handleToken(args shared.HandlerRequest) shared.HandlerResponse {
	if !strings.EqualFold(args.Method, "POST") {
		return o.tokenError("invalid_request", "Only POST method is allowed")
	}

	// Parse form data from request body
	formData, err := parseFormData(args.Body)
	if err != nil {
		return o.tokenError("invalid_request", "Invalid form data")
	}

	grantType := formData.Get("grant_type")
	if grantType != "authorization_code" {
		return o.tokenError("unsupported_grant_type", "Only authorization_code grant type is supported")
	}

	code := formData.Get("code")
	redirectURI := formData.Get("redirect_uri")
	codeVerifier := formData.Get("code_verifier")

	// Client authentication: support both client_secret_post and client_secret_basic
	var clientID, clientSecret string

	// Try client_secret_post first (credentials in form data)
	clientID = formData.Get("client_id")
	clientSecret = formData.Get("client_secret")

	// If no client_id in form data, try client_secret_basic (Authorization header)
	if clientID == "" {
		authHeader := args.Headers["authorization"]
		if authHeader == "" {
			authHeader = args.Headers["Authorization"]
		}

		if authHeader != "" {
			basicClientID, basicClientSecret, ok := parseBasicAuth(authHeader)
			if ok {
				clientID = basicClientID
				clientSecret = basicClientSecret
			}
		}
	}

	// Validate required parameters
	if code == "" {
		return o.tokenError("invalid_request", "code is required")
	}

	if clientID == "" {
		return o.tokenError("invalid_client", "client_id is required (either in form data or Authorization header)")
	}

	if redirectURI == "" {
		return o.tokenError("invalid_request", "redirect_uri is required")
	}

	// Get and validate authorization code
	o.mutex.RLock()
	authCode, exists := o.codes[code]
	o.mutex.RUnlock()

	if !exists {
		return o.tokenError("invalid_grant", "Invalid authorization code")
	}

	// Check if code has expired
	if time.Now().After(authCode.ExpiresAt) {
		o.mutex.Lock()
		delete(o.codes, code)
		o.mutex.Unlock()
		return o.tokenError("invalid_grant", "Authorization code has expired")
	}

	// Validate client
	client := o.config.GetClient(clientID)
	if client == nil {
		return o.tokenError("invalid_client", "Unknown client")
	}

	// Validate client credentials if client secret is configured
	if client.ClientSecret != "" && client.ClientSecret != clientSecret {
		return o.tokenError("invalid_client", "Invalid client credentials")
	}

	// Validate that the code belongs to this client
	if authCode.ClientID != clientID {
		return o.tokenError("invalid_grant", "Authorization code does not belong to this client")
	}

	// Validate redirect URI
	if authCode.RedirectURI != redirectURI {
		return o.tokenError("invalid_grant", "redirect_uri does not match")
	}

	// Validate PKCE if present
	if authCode.CodeChallenge != "" {
		if err := validatePKCE(authCode.CodeChallenge, authCode.ChallengeMethod, codeVerifier); err != nil {
			return o.tokenError("invalid_grant", fmt.Sprintf("PKCE validation failed: %v", err))
		}
	}

	// Get user
	user := o.config.GetUser(authCode.UserID)
	if user == nil {
		return o.tokenError("server_error", "User not found")
	}

	// Generate access token
	accessToken := o.generateAccessToken()
	now := time.Now()
	expiresIn := 3600 // 1 hour

	// Store access token
	o.mutex.Lock()
	o.tokens[accessToken] = &AccessToken{
		Token:     accessToken,
		UserID:    authCode.UserID,
		ClientID:  clientID,
		Scope:     authCode.Scope,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(expiresIn) * time.Second),
	}
	// Remove the authorization code (one-time use)
	delete(o.codes, code)
	o.mutex.Unlock()

	// Create ID token if openid scope is present
	var idToken string
	scopes := strings.Split(authCode.Scope, " ")
	if containsScope(scopes, "openid") {
		idToken, err = o.generateIDToken(user, clientID, authCode.Nonce, scopes, now, expiresIn)
		if err != nil {
			o.logger.Error("failed to generate ID token", "error", err)
			return o.tokenError("server_error", "Failed to generate ID token")
		}
	}

	response := TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		IDToken:     idToken,
		Scope:       authCode.Scope,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return o.tokenError("server_error", "Failed to encode response")
	}

	return shared.HandlerResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Cache-Control": "no-store",
			"Pragma":        "no-cache",
		},
		Body: responseBody,
	}
}

func (o *OIDCServer) generateIDToken(user *User, clientID, nonce string, scopes []string, issuedAt time.Time, expiresIn int) (string, error) {
	// Create JWT claims
	claims := jwt.MapClaims{
		"iss": o.serverURL,                                                 // issuer (use configured server URL)
		"sub": user.Username,                                               // subject
		"aud": clientID,                                                    // audience
		"exp": issuedAt.Add(time.Duration(expiresIn) * time.Second).Unix(), // expiration
		"iat": issuedAt.Unix(),                                             // issued at
	}

	// Add nonce if present
	if nonce != "" {
		claims["nonce"] = nonce
	}

	// Add user claims based on scopes
	userClaims := o.getUserClaims(user, scopes)
	for key, value := range userClaims {
		if key != "sub" { // sub is already set above
			claims[key] = value
		}
	}

	// Create token with appropriate signing method
	var token *jwt.Token
	var signedToken string
	var err error

	if o.config.JWTConfig != nil && o.config.JWTConfig.Algorithm == "RS256" {
		// Use RS256 with RSA private key
		token = jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		// Add key ID to header for RS256
		token.Header["kid"] = o.config.JWTConfig.KeyID
		signedToken, err = token.SignedString(o.privateKey)
	} else {
		// Default to HS256 with HMAC secret
		token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err = token.SignedString(o.jwtSecret)
	}

	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	return signedToken, nil
}

func (o *OIDCServer) validateAccessToken(tokenString string) (*AccessToken, error) {
	o.mutex.RLock()
	token, exists := o.tokens[tokenString]
	o.mutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("invalid token")
	}

	if time.Now().After(token.ExpiresAt) {
		o.mutex.Lock()
		delete(o.tokens, tokenString)
		o.mutex.Unlock()
		return nil, fmt.Errorf("token expired")
	}

	return token, nil
}

func (o *OIDCServer) tokenError(errorCode, errorDescription string) shared.HandlerResponse {
	o.logger.Error("token error", "error", errorCode, "description", errorDescription)

	errorResponse := TokenErrorResponse{
		Error:            errorCode,
		ErrorDescription: errorDescription,
	}

	body, _ := json.Marshal(errorResponse)

	return shared.HandlerResponse{
		StatusCode: 400,
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Cache-Control": "no-store",
			"Pragma":        "no-cache",
		},
		Body: body,
	}
}

func containsScope(scopes []string, scope string) bool {
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func parseBasicAuth(header string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(header, prefix) {
		return "", "", false
	}

	// Extract the base64 encoded credentials
	encoded := header[len(prefix):]
	if encoded == "" {
		return "", "", false
	}

	// Decode from base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", false
	}

	// Split on first colon to separate username and password
	credentials := string(decoded)
	colonIndex := strings.IndexByte(credentials, ':')
	if colonIndex == -1 {
		return "", "", false
	}

	return credentials[:colonIndex], credentials[colonIndex+1:], true
}
