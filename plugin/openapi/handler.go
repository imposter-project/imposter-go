package openapi

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"net/http"
)

// HandleRequest handles incoming HTTP requests
func (h *PluginHandler) HandleRequest(
	r *http.Request,
	requestStore *store.Store,
	responseState *response.ResponseState,
	respProc response.Processor,
) {
	// TODO validate request against OpenAPI spec

	wrapped := func(reqMatcher *config.RequestMatcher, rs *response.ResponseState, r *http.Request, resp *config.Response, requestStore *store.Store) {
		h.processResponse(reqMatcher, rs, r, resp, requestStore, respProc)
	}

	h.restPluginHandler.HandleRequest(r, requestStore, responseState, wrapped)
}
