package rest

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"net/http"
)

// processResponse handles preparing the response state
func (h *PluginHandler) processResponse(
	reqMatcher *config.RequestMatcher,
	rs *response.ResponseState,
	r *http.Request,
	resp *config.Response,
	requestStore *store.Store,
	respProc response.Processor,
) {
	// Standard response processor
	respProc(reqMatcher, rs, r, resp, requestStore)
}
