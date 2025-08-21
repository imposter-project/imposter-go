package main

import (
	"fmt"
	"html/template"
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

	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Sign In - OIDC Server</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            margin: 0;
            padding: 20px;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .login-container {
            background: white;
            padding: 40px;
            border-radius: 10px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.1);
            width: 100%;
            max-width: 400px;
        }
        .login-header {
            text-align: center;
            margin-bottom: 30px;
        }
        .login-header h1 {
            color: #333;
            margin: 0;
            font-size: 28px;
        }
        .login-header p {
            color: #666;
            margin: 10px 0 0 0;
        }
        .form-group {
            margin-bottom: 20px;
        }
        .form-group label {
            display: block;
            margin-bottom: 5px;
            color: #333;
            font-weight: 500;
        }
        .form-group input {
            width: 100%;
            padding: 12px;
            border: 2px solid #e1e5e9;
            border-radius: 5px;
            font-size: 16px;
            transition: border-color 0.3s;
            box-sizing: border-box;
        }
        .form-group input:focus {
            outline: none;
            border-color: #667eea;
        }
        .btn {
            width: 100%;
            padding: 12px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 5px;
            font-size: 16px;
            cursor: pointer;
            transition: transform 0.2s;
        }
        .btn:hover {
            transform: translateY(-2px);
        }
        .error {
            background: #fee;
            color: #c33;
            padding: 10px;
            border-radius: 5px;
            margin-bottom: 20px;
            border: 1px solid #fcc;
        }
        .client-info {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 5px;
            margin-bottom: 20px;
            text-align: center;
        }
        .client-info p {
            margin: 0;
            color: #666;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-header">
            <h1>Sign In</h1>
            <p>OIDC Authorization Server</p>
        </div>
        
        <div class="client-info">
            <p>Application <strong>{{.ClientID}}</strong> is requesting access to your account</p>
        </div>
        
        {{if .Error}}
        <div class="error">{{.Error}}</div>
        {{end}}
        
        <form method="POST" action="/oidc/authorize">
            <input type="hidden" name="session_id" value="{{.SessionID}}">
            
            <div class="form-group">
                <label for="username">Username</label>
                <input type="text" id="username" name="username" required autofocus>
            </div>
            
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required>
            </div>
            
            <button type="submit" class="btn">Sign In</button>
        </form>
    </div>
</body>
</html>`

	t, err := template.New("login").Parse(tmpl)
	if err != nil {
		o.logger.Error("failed to parse login template", "error", err)
		return shared.HandlerResponse{StatusCode: 500, Body: []byte("Internal Server Error")}
	}

	data := struct {
		SessionID string
		ClientID  string
		Error     string
	}{
		SessionID: sessionID,
		ClientID:  clientID,
		Error:     errorMessage,
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
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
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Error - OIDC Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .error { background: #fee; color: #c33; padding: 20px; border-radius: 5px; border: 1px solid #fcc; }
        .error h1 { margin: 0 0 10px 0; }
        .error p { margin: 0; }
    </style>
</head>
<body>
    <div class="error">
        <h1>Error: %s</h1>
        <p>%s</p>
    </div>
</body>
</html>`, errorCode, errorDescription)

	return shared.HandlerResponse{
		StatusCode: 400,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		Body: []byte(body),
	}
}

func (o *OIDCServer) redirectError(redirectURI, errorCode, errorDescription, state string) shared.HandlerResponse {
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
