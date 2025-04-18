package rest

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
)

// processResponse handles preparing the response state
func (h *PluginHandler) processResponse(
	exch *exchange.Exchange,
	reqMatcher *config.RequestMatcher,
	resp *config.Response,
	respProc response.Processor,
) {
	// Standard response processor
	respProc(exch, reqMatcher, resp)
}
