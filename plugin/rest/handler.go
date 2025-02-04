package rest

import (
	"net/http"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
)

// HandleRequest processes incoming REST API requests
func (h *PluginHandler) HandleRequest(
	r *http.Request,
	requestStore *store.Store,
	responseState *response.ResponseState,
	respProc response.Processor,
) {
	body, err := matcher.GetRequestBody(r)
	if err != nil {
		responseState.StatusCode = http.StatusBadRequest
		responseState.Body = []byte("Failed to read request body")
		responseState.Handled = true
		return
	}

	// Get system XML namespaces
	var systemNamespaces map[string]string
	if h.config.System != nil {
		systemNamespaces = h.config.System.XMLNamespaces
	}

	// Process interceptors first
	for _, interceptorCfg := range h.config.Interceptors {
		score, _ := matcher.CalculateMatchScore(&interceptorCfg.RequestMatcher, r, body, systemNamespaces, h.imposterConfig, requestStore)
		if score > 0 {
			logger.Infof("matched interceptor - method:%s, path:%s", r.Method, r.URL.Path)
			if interceptorCfg.Capture != nil {
				capture.CaptureRequestData(h.imposterConfig, interceptorCfg.Capture, r, body, requestStore)
			}
			if interceptorCfg.Response != nil {
				h.processResponse(&interceptorCfg.RequestMatcher, responseState, r, interceptorCfg.Response, requestStore, respProc)
			}
			if !interceptorCfg.Continue {
				responseState.Handled = true
				return // Short-circuit if interceptor continue is false
			}
		}
	}

	var matches []matcher.MatchResult
	for _, res := range h.config.Resources {
		score, isWildcard := matcher.CalculateMatchScore(&res.RequestMatcher, r, body, systemNamespaces, h.imposterConfig, requestStore)
		if score > 0 {
			matches = append(matches, matcher.MatchResult{Resource: &res, Score: score, Wildcard: isWildcard, RuntimeGenerated: res.RuntimeGenerated})
		}
	}

	if len(matches) == 0 {
		return // Let the main handler deal with no matches
	}

	// Find the best match
	best, tie := matcher.FindBestMatch(matches)
	if tie {
		logger.Warnf("multiple equally specific matches, using the first")
	}

	// Capture request data
	capture.CaptureRequestData(h.imposterConfig, best.Resource.Capture, r, body, requestStore)

	// Process the response
	h.processResponse(&best.Resource.RequestMatcher, responseState, r, &best.Resource.Response, requestStore, respProc)
	responseState.Handled = true
}
