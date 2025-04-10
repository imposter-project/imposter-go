package openapi

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"net/http"
)

// HandleRequest handles incoming HTTP requests
func (h *PluginHandler) HandleRequest(
	r *http.Request,
	requestStore *store.Store,
	responseState *response.ResponseState,
	respProc response.Processor,
) {
	// If request validation is enabled, validate against OpenAPI spec
	if h.config.Validation != nil && h.config.Validation.Request {
		logger.Debugf("Validating request %s %s against OpenAPI spec", r.Method, r.URL.Path)
		valid, validationErrors := h.openApiParser.ValidateRequest(r)
		if !valid {
			logger.Warnf("Request validation failed for %s %s", r.Method, r.URL.Path)
			for _, err := range validationErrors {
				logger.Warnf("  - %s", err.Message)
			}
			// We continue processing even if validation fails
		}
	}

	wrapped := func(reqMatcher *config.RequestMatcher, rs *response.ResponseState, r *http.Request, resp *config.Response, requestStore *store.Store) {
		// Process the response
		h.processResponse(reqMatcher, rs, r, resp, requestStore, respProc)

		// If response validation is enabled, validate the processed response
		// Note that the response has already been sent at this point
		if h.config.Validation != nil && h.config.Validation.Response && rs.Handled {
			logger.Debugf("Validating response for %s %s with status %d", r.Method, r.URL.Path, rs.StatusCode)
			valid, validationErrors := h.openApiParser.ValidateResponse(rs)
			if !valid {
				logger.Warnf("Response validation failed for %s %s", r.Method, r.URL.Path)
				for _, err := range validationErrors {
					logger.Warnf("  - %s", err.Message)
				}
			}
		}
	}

	h.restPluginHandler.HandleRequest(r, requestStore, responseState, wrapped)
}
