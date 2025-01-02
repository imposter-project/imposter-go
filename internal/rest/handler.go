package rest

import (
	"fmt"
	"net/http"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/plugin"
	"github.com/imposter-project/imposter-go/internal/store"
)

// Handler handles REST API requests
type Handler struct {
	config         *config.Config
	configDir      string
	imposterConfig *config.ImposterConfig
}

// NewHandler creates a new REST handler
func NewHandler(cfg *config.Config, configDir string, imposterConfig *config.ImposterConfig) (*Handler, error) {
	return &Handler{
		config:         cfg,
		configDir:      configDir,
		imposterConfig: imposterConfig,
	}, nil
}

// HandleRequest processes incoming REST API requests
func (h *Handler) HandleRequest(r *http.Request, requestStore store.Store, responseState *plugin.ResponseState) {
	body, err := plugin.GetRequestBody(r)
	if err != nil {
		responseState.StatusCode = http.StatusBadRequest
		responseState.Body = []byte("Failed to read request body")
		responseState.Handled = true
		return
	}

	// Process interceptors first
	for _, interceptor := range h.config.Interceptors {
		score, isWildcard := plugin.CalculateMatchScore(&interceptor.RequestMatcher, r, body)
		if score > 0 {
			fmt.Printf("Matched interceptor - method:%s, path:%s, wildcard:%v\n",
				r.Method, r.URL.Path, isWildcard)
			// Process the interceptor
			if !h.processInterceptor(responseState, r, body, interceptor, requestStore) {
				responseState.Handled = true
				return // Short-circuit if interceptor responded and continue is false
			}
		}
	}

	var matches []plugin.MatchResult
	for _, res := range h.config.Resources {
		score, isWildcard := plugin.CalculateMatchScore(&res.RequestMatcher, r, body)
		if score > 0 {
			matches = append(matches, plugin.MatchResult{Resource: &res, Score: score, Wildcard: isWildcard})
		}
	}

	if len(matches) == 0 {
		return // Let the main handler deal with no matches
	}

	// Find the best match
	best, tie := plugin.FindBestMatch(matches)
	if tie {
		fmt.Printf("Warning: multiple equally specific matches. Using the first.\n")
	}

	// Capture request data
	capture.CaptureRequestData(h.imposterConfig, *best.Resource, r, body, requestStore)

	// Process the response
	h.processResponse(responseState, r, best.Resource.Response, requestStore)
	responseState.Handled = true
}

// processInterceptor handles an interceptor and returns true if request processing should continue
func (h *Handler) processInterceptor(rs *plugin.ResponseState, r *http.Request, body []byte, interceptor config.Interceptor, requestStore store.Store) bool {
	// Capture request data if specified
	if interceptor.Capture != nil {
		capture.CaptureRequestData(h.imposterConfig, config.Resource{
			RequestMatcher: config.RequestMatcher{
				Capture: interceptor.Capture,
			},
		}, r, body, requestStore)
	}

	// If the interceptor has a response and continue is false, send the response and stop processing
	if interceptor.Response != nil {
		h.processResponse(rs, r, *interceptor.Response, requestStore)
	}

	return interceptor.Continue
}

// processResponse handles preparing the response state
func (h *Handler) processResponse(rs *plugin.ResponseState, r *http.Request, response config.Response, requestStore store.Store) {
	plugin.ProcessResponse(rs, r, response, h.configDir, requestStore, h.imposterConfig)
}
