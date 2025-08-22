package main

import (
	"encoding/json"
	"strings"

	"github.com/imposter-project/imposter-go/external/shared"
)

type UserinfoErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func (o *OIDCServer) handleUserInfo(args shared.HandlerRequest) shared.HandlerResponse {
	if !strings.EqualFold(args.Method, "GET") && !strings.EqualFold(args.Method, "POST") {
		return o.userinfoError("invalid_request", "Only GET and POST methods are allowed", 405)
	}

	// Extract Bearer token from Authorization header
	authHeader := args.Headers["authorization"]
	if authHeader == "" {
		authHeader = args.Headers["Authorization"]
	}

	if authHeader == "" {
		return o.userinfoError("invalid_token", "Authorization header is required", 401)
	}

	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return o.userinfoError("invalid_token", "Invalid authorization header format", 401)
	}

	accessToken := strings.TrimPrefix(authHeader, bearerPrefix)
	if accessToken == "" {
		return o.userinfoError("invalid_token", "Access token is required", 401)
	}

	// Validate access token
	token, err := o.validateAccessToken(accessToken)
	if err != nil {
		return o.userinfoError("invalid_token", err.Error(), 401)
	}

	// Get user
	user := o.config.GetUser(token.UserID)
	if user == nil {
		return o.userinfoError("server_error", "User not found", 500)
	}

	// Parse scopes
	scopes := strings.Split(token.Scope, " ")

	// Build userinfo response based on scopes
	userClaims := o.getUserClaims(user, scopes)

	// Always include sub claim
	if _, exists := userClaims["sub"]; !exists {
		userClaims["sub"] = user.Username
	}

	responseBody, err := json.Marshal(userClaims)
	if err != nil {
		o.logger.Error("failed to marshal userinfo response", "error", err)
		return o.userinfoError("server_error", "Failed to encode response", 500)
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

func (o *OIDCServer) userinfoError(errorCode, errorDescription string, statusCode int) shared.HandlerResponse {
	o.logger.Error("userinfo error", "error", errorCode, "description", errorDescription)

	errorResponse := UserinfoErrorResponse{
		Error:            errorCode,
		ErrorDescription: errorDescription,
	}

	body, _ := json.Marshal(errorResponse)

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// Add WWW-Authenticate header for 401 responses
	if statusCode == 401 {
		headers["WWW-Authenticate"] = `Bearer realm="oidc-server"`
	}

	return shared.HandlerResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
	}
}
