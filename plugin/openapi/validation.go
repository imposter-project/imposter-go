package openapi

import (
	"encoding/json"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/pb33f/libopenapi-validator/errors"
	"net/http"
)

// ValidationErrorResponse represents a response for validation errors
type ValidationErrorResponse struct {
	Message string                   `json:"message"`
	Errors  []ValidationErrorDetails `json:"errors"`
}

// ValidationErrorDetails contains details about a validation error
type ValidationErrorDetails struct {
	Message   string `json:"message"`
	ErrorType string `json:"errorType,omitempty"`
}

// validateRequest validates the request against the OpenAPI spec and returns if the request should continue processing
func (h *PluginHandler) validateRequest(
	r *http.Request,
	responseState *response.ResponseState,
) bool {
	// If validation is not enabled, continue processing
	if h.config.Validation == nil || !h.config.Validation.IsRequestValidationEnabled() {
		logger.Tracef("request validation is disabled, skipping validation for %s %s", r.Method, r.URL.Path)
		return true
	}

	behaviour := h.config.Validation.GetRequestBehaviour()
	logger.Debugf("validating request %s %s against OpenAPI spec (behaviour: %s)", r.Method, r.URL.Path, behaviour)

	valid, validationErrors := h.openApiParser.ValidateRequest(r)
	if valid {
		logger.Tracef("request %s %s is valid", r.Method, r.URL.Path)
		return true
	}

	// Only process if validationErrors is not nil
	if validationErrors == nil || len(validationErrors) == 0 {
		logger.Warnf("request validation failed but no validation errors were returned")
		return true
	}

	switch behaviour {
	case config.ValidationBehaviourFail:
		// Fail the request by setting an error response
		logger.Warnf("request validation failed for %s %s - failing request", r.Method, r.URL.Path)
		for _, err := range validationErrors {
			logger.Warnf("  - %s", err.Message)
		}

		// Create validation errors response
		responseState.StatusCode = http.StatusBadRequest
		responseState.Body = createValidationErrorResponse(validationErrors)
		if responseState.Headers == nil {
			responseState.Headers = make(map[string]string)
		}
		responseState.Headers["Content-Type"] = "application/json"
		responseState.Handled = true
		return false // Stop further processing

	case config.ValidationBehaviourLog:
		// Just log the validation errors
		logger.Warnf("request validation failed for %s %s - logging only", r.Method, r.URL.Path)
		for _, err := range validationErrors {
			logger.Warnf("  - %s", err.Message)
		}
		return true // Continue processing

	default: // ValidationBehaviourIgnore
		// Do nothing, just ignore validation errors
		logger.Debugf("request validation failed for %s %s - ignoring", r.Method, r.URL.Path)
		return true
	}
}

// createValidationErrorResponse creates a JSON response for validation errors
func createValidationErrorResponse(validationErrors []*errors.ValidationError) []byte {
	details := make([]ValidationErrorDetails, 0, len(validationErrors))

	for _, err := range validationErrors {
		detail := ValidationErrorDetails{
			Message:   err.Message,
			ErrorType: string(err.ValidationType),
		}
		details = append(details, detail)
	}

	resp := ValidationErrorResponse{
		Message: "OpenAPI request validation failed",
		Errors:  details,
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		logger.Errorf("failed to marshal validation errors: %v", err)
		return []byte(`{"message":"OpenAPI request validation failed"}`)
	}

	return jsonData
}
