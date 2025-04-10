package openapi

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"net/http"
)

func (h *PluginHandler) HandleRequest(
	r *http.Request,
	requestStore *store.Store,
	responseState *response.ResponseState,
	respProc response.Processor,
) {
	// Validate request against OpenAPI spec
	if !h.validateRequest(r, responseState) {
		return // Stop processing if validation failed
	}

	wrapped := func(reqMatcher *config.RequestMatcher, rs *response.ResponseState, r *http.Request, resp *config.Response, requestStore *store.Store) {
		// Process the response
		h.processResponse(reqMatcher, rs, r, resp, requestStore, respProc)

		// If response validation is enabled, validate the processed response
		// Note that the response has already been sent at this point
		if h.config.Validation != nil && h.config.Validation.IsResponseValidationEnabled() && rs.Handled {
			logger.Debugf("response validation not fully implemented yet")
		}
	}

	h.restPluginHandler.HandleRequest(r, requestStore, responseState, wrapped)
}
