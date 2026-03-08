package main

import (
	"net/url"
	"strings"
	"time"

	"github.com/imposter-project/imposter-go/external/shared"
)

func (o *OIDCServer) handleAuthorize(args shared.HandlerRequest) shared.HandlerResponse {
	switch strings.ToUpper(args.Method) {
	case "GET":
		return o.handleAuthorizeGet(args)
	case "POST":
		return o.handleAuthorizePost(args)
	default:
		return shared.HandlerResponse{StatusCode: 405, Body: []byte("Method Not Allowed")}
	}
}

func (o *OIDCServer) handleAuthorizeGet(args shared.HandlerRequest) shared.HandlerResponse {
	// Parse query parameters
	if len(args.Query) == 0 {
		return o.renderError("invalid_request", "Missing query parameters")
	}

	// Validate required parameters
	clientID := args.Query.Get("client_id")
	redirectURI := args.Query.Get("redirect_uri")
	responseType := args.Query.Get("response_type")
	scope := args.Query.Get("scope")
	state := args.Query.Get("state")
	nonce := args.Query.Get("nonce")
	codeChallenge := args.Query.Get("code_challenge")
	codeChallengeMethod := args.Query.Get("code_challenge_method")

	if clientID == "" {
		return o.renderError("invalid_request", "client_id is required")
	}

	if redirectURI == "" {
		return o.renderError("invalid_request", "redirect_uri is required")
	}

	if responseType != "code" {
		return o.redirectError(redirectURI, "unsupported_response_type", "Only 'code' response type is supported", state)
	}

	if scope == "" || !strings.Contains(scope, "openid") {
		return o.redirectError(redirectURI, "invalid_scope", "openid scope is required", state)
	}

	// Validate client
	client := o.config.GetClient(clientID)
	if client == nil {
		return o.redirectError(redirectURI, "invalid_client", "Unknown client", state)
	}

	if !client.IsValidRedirectURI(redirectURI) {
		return o.renderError("invalid_request", "Invalid redirect_uri")
	}

	// Validate PKCE if present
	if err := validateCodeChallenge(codeChallenge, codeChallengeMethod); err != nil {
		return o.redirectError(redirectURI, "invalid_request", err.Error(), state)
	}

	// Create authorization session
	sessionID := o.generateSessionID()
	session := &AuthSession{
		ID:              sessionID,
		ClientID:        clientID,
		RedirectURI:     redirectURI,
		Scope:           scope,
		State:           state,
		Nonce:           nonce,
		CodeChallenge:   codeChallenge,
		ChallengeMethod: codeChallengeMethod,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(10 * time.Minute),
	}

	o.mutex.Lock()
	o.sessions[sessionID] = session
	o.mutex.Unlock()

	// Render login form
	return o.renderLoginForm(sessionID, client.ClientID)
}

func (o *OIDCServer) handleAuthorizePost(args shared.HandlerRequest) shared.HandlerResponse {
	// Parse form data
	formData, err := parseFormData(args.Body)
	if err != nil {
		return o.renderError("invalid_request", "Invalid form data")
	}

	sessionID := formData.Get("session_id")
	username := formData.Get("username")
	password := formData.Get("password")

	if sessionID == "" {
		return o.renderError("invalid_request", "session_id is required")
	}

	// Get session
	o.mutex.RLock()
	session, exists := o.sessions[sessionID]
	o.mutex.RUnlock()

	if !exists || time.Now().After(session.ExpiresAt) {
		return o.renderError("invalid_request", "Invalid or expired session")
	}

	// Authenticate user
	user := o.authenticateUser(username, password)
	if user == nil {
		return o.renderLoginForm(sessionID, session.ClientID, "Invalid username or password")
	}

	// Generate authorization code
	code := o.generateAuthCode()
	authCode := &AuthCode{
		Code:            code,
		ClientID:        session.ClientID,
		RedirectURI:     session.RedirectURI,
		UserID:          user.Username,
		Scope:           session.Scope,
		Nonce:           session.Nonce,
		CodeChallenge:   session.CodeChallenge,
		ChallengeMethod: session.ChallengeMethod,
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(10 * time.Minute),
	}

	// Store authorization code and clean up session
	o.mutex.Lock()
	o.codes[code] = authCode
	delete(o.sessions, sessionID)
	o.mutex.Unlock()

	// Redirect back to client with authorization code
	redirectURL, err := url.Parse(session.RedirectURI)
	if err != nil {
		return o.renderError("server_error", "Invalid redirect URI")
	}

	values := redirectURL.Query()
	values.Set("code", code)
	if session.State != "" {
		values.Set("state", session.State)
	}
	redirectURL.RawQuery = values.Encode()

	return shared.HandlerResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Location": redirectURL.String(),
		},
	}
}

func (o *OIDCServer) renderLoginForm(sessionID, clientID string, errorMsg ...string) shared.HandlerResponse {
	var errorMessage string
	if len(errorMsg) > 0 {
		errorMessage = errorMsg[0]
	}

	data := struct {
		SessionID  string
		ClientID   string
		PathPrefix string
		Error      string
	}{
		SessionID:  sessionID,
		ClientID:   clientID,
		PathPrefix: o.pathPrefix,
		Error:      errorMessage,
	}

	var buf strings.Builder
	if err := loginTemplate.Execute(&buf, data); err != nil {
		o.logger.Error("failed to execute login template", "error", err)
		return shared.HandlerResponse{StatusCode: 500, Body: []byte("Internal Server Error")}
	}

	return shared.HandlerResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		Body: []byte(buf.String()),
	}
}

func (o *OIDCServer) renderError(errorCode, errorDescription string) shared.HandlerResponse {
	o.logger.Error("OIDC error", "code", errorCode, "description", errorDescription)

	data := struct {
		ErrorCode        string
		ErrorDescription string
	}{
		ErrorCode:        errorCode,
		ErrorDescription: errorDescription,
	}

	var buf strings.Builder
	if err := errorTemplate.Execute(&buf, data); err != nil {
		o.logger.Error("failed to execute error template", "error", err)
		return shared.HandlerResponse{StatusCode: 500, Body: []byte("Internal Server Error")}
	}

	return shared.HandlerResponse{
		StatusCode: 400,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		Body: []byte(buf.String()),
	}
}

func (o *OIDCServer) redirectError(redirectURI, errorCode, errorDescription, state string) shared.HandlerResponse {
	o.logger.Error("OIDC error", "code", errorCode, "description", errorDescription)

	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		return o.renderError("server_error", "Invalid redirect URI")
	}

	values := redirectURL.Query()
	values.Set("error", errorCode)
	values.Set("error_description", errorDescription)
	if state != "" {
		values.Set("state", state)
	}
	redirectURL.RawQuery = values.Encode()

	return shared.HandlerResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Location": redirectURL.String(),
		},
	}
}
