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
	respPreprocessor response.Processor,
) {
	if respPreprocessor != nil {
		respPreprocessor(reqMatcher, rs, r, resp, requestStore)
	}
	response.ProcessResponse(reqMatcher, rs, r, resp, h.configDir, requestStore, h.imposterConfig)
}
