package openapi

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

func (h *PluginHandler) HandleRequest(
	exch *exchange.Exchange,
	respProc response.Processor,
) {
	// Validate request against OpenAPI spec
	if !h.validateRequest(exch.Request.Request, exch.ResponseState) {
		return // Stop processing if validation failed
	}

	wrapped := func(exch *exchange.Exchange, reqMatcher *config.RequestMatcher, resp *config.Response) {
		// Process the response
		h.processResponse(exch, reqMatcher, resp, respProc)

		// If response validation is enabled, validate the processed response
		// Note that the response has already been sent at this point
		if h.config.Validation != nil && h.config.Validation.IsResponseValidationEnabled() && exch.ResponseState.Handled {
			logger.Debugf("response validation not fully implemented yet")
		}
	}

	h.restPluginHandler.HandleRequest(exch, wrapped)
}
