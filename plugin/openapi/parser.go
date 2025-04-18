package openapi

import (
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/pkg/logger"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/errors"
	"net/http"
)

// openAPIParser is a base struct that provides common functionality for OpenAPI parsers
type openAPIParser struct {
	validator *validator.Validator
}

// ValidateRequest validates an HTTP request against the OpenAPI specification
func (p *openAPIParser) ValidateRequest(req *http.Request) (bool, []*errors.ValidationError) {
	if p.validator == nil {
		logger.Debugf("no validator available for request validation")
		return true, nil
	}

	valid, validationErrors := (*p.validator).ValidateHttpRequest(req)
	if !valid {
		for _, err := range validationErrors {
			logger.Warnf("request validation error: %s", err.Message)
		}
	}

	return valid, validationErrors
}

// ValidateResponse is a placeholder for response validation against the OpenAPI specification
// Currently not fully implemented
func (p *openAPIParser) ValidateResponse(rs *exchange.ResponseState) (bool, []*errors.ValidationError) {
	logger.Debugf("response validation not supported")
	return true, nil
}
