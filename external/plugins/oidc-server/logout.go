package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/imposter-project/imposter-go/external/shared"
)

func (o *OIDCServer) handleLogout(args shared.HandlerRequest) shared.HandlerResponse {
	switch strings.ToUpper(args.Method) {
	case "GET":
		return o.handleLogoutRequest(args.Query)
	case "POST":
		formData, err := parseFormData(args.Body)
		if err != nil {
			return o.renderError("invalid_request", "Invalid form data")
		}
		return o.handleLogoutRequest(formData)
	default:
		return shared.HandlerResponse{StatusCode: 405, Body: []byte("Method Not Allowed")}
	}
}

func (o *OIDCServer) handleLogoutRequest(params url.Values) shared.HandlerResponse {
	idTokenHint := params.Get("id_token_hint")
	postLogoutRedirectURI := params.Get("post_logout_redirect_uri")
	clientID := params.Get("client_id")
	state := params.Get("state")

	// Extract client ID and user ID from id_token_hint if provided
	var tokenClientID, userID string
	if idTokenHint != "" {
		var err error
		tokenClientID, userID, err = o.parseIDTokenHint(idTokenHint)
		if err != nil {
			o.logger.Warn("failed to parse id_token_hint", "error", err)
		}
	}

	// Explicit client_id parameter takes precedence over token claim
	if clientID == "" {
		clientID = tokenClientID
	}

	// Validate post_logout_redirect_uri if provided
	if postLogoutRedirectURI != "" {
		if clientID == "" {
			return o.renderError("invalid_request", "client_id or id_token_hint is required when post_logout_redirect_uri is provided")
		}

		client := o.config.GetClient(clientID)
		if client == nil {
			return o.renderError("invalid_request", "Unknown client")
		}

		if !client.IsValidPostLogoutRedirectURI(postLogoutRedirectURI) {
			return o.renderError("invalid_request", "Invalid post_logout_redirect_uri")
		}
	}

	// Clear user state
	o.clearUserState(userID)

	// Redirect or show confirmation
	if postLogoutRedirectURI != "" {
		return o.redirectAfterLogout(postLogoutRedirectURI, state)
	}
	return o.renderLogoutConfirmation()
}

func (o *OIDCServer) parseIDTokenHint(tokenString string) (clientID, sub string, err error) {
	// Parse without validating expiry — the spec requires accepting expired tokens
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	token, err := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method matches server configuration
		if o.config.JWTConfig != nil && o.config.JWTConfig.Algorithm == "HS256" {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return o.jwtSecret, nil
		}
		// Default to RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return o.publicKey, nil
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to parse id_token_hint: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("failed to extract claims from id_token_hint")
	}

	// Verify issuer matches this server
	iss, _ := claims["iss"].(string)
	if iss != o.serverURL {
		return "", "", fmt.Errorf("issuer mismatch: expected %s, got %s", o.serverURL, iss)
	}

	// Extract audience (client ID) and subject (user ID)
	clientID, _ = claims["aud"].(string)
	sub, _ = claims["sub"].(string)

	return clientID, sub, nil
}

func (o *OIDCServer) clearUserState(userID string) {
	if userID == "" {
		return
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	for key, token := range o.tokens {
		if token.UserID == userID {
			delete(o.tokens, key)
		}
	}
	for key, code := range o.codes {
		if code.UserID == userID {
			delete(o.codes, key)
		}
	}
}

func (o *OIDCServer) redirectAfterLogout(uri, state string) shared.HandlerResponse {
	redirectURL, err := url.Parse(uri)
	if err != nil {
		return o.renderError("server_error", "Invalid post_logout_redirect_uri")
	}

	if state != "" {
		values := redirectURL.Query()
		values.Set("state", state)
		redirectURL.RawQuery = values.Encode()
	}

	return shared.HandlerResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Location":      redirectURL.String(),
			"Cache-Control": "no-store",
		},
	}
}

func (o *OIDCServer) renderLogoutConfirmation() shared.HandlerResponse {
	body := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Signed Out - OIDC Server</title>
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
        .logout-container {
            background: white;
            padding: 40px;
            border-radius: 10px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.1);
            width: 100%;
            max-width: 400px;
            text-align: center;
        }
        .logout-container h1 {
            color: #333;
            margin: 0 0 15px 0;
            font-size: 28px;
        }
        .logout-container p {
            color: #666;
            margin: 0;
            font-size: 16px;
        }
    </style>
</head>
<body>
    <div class="logout-container">
        <h1>Signed Out</h1>
        <p>You have been signed out of the OIDC Authorisation Server.</p>
    </div>
</body>
</html>`

	return shared.HandlerResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":  "text/html; charset=utf-8",
			"Cache-Control": "no-store",
		},
		Body: []byte(body),
	}
}
