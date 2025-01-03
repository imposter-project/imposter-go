package rest

import (
	"fmt"
	"net/http"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	commonInterceptor "github.com/imposter-project/imposter-go/internal/interceptor"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/response"
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
func (h *Handler) HandleRequest(r *http.Request, requestStore store.Store, responseState *response.ResponseState) {
	body, err := matcher.GetRequestBody(r)
	if err != nil {
		responseState.StatusCode = http.StatusBadRequest
		responseState.Body = []byte("Failed to read request body")
		responseState.Handled = true
		return
	}

	// Process interceptors first
	for _, interceptorCfg := range h.config.Interceptors {
		score, isWildcard := matcher.CalculateMatchScore(&interceptorCfg.RequestMatcher, r, body)
		if score > 0 {
			fmt.Printf("Matched interceptor - method:%s, path:%s, wildcard:%v\n",
				r.Method, r.URL.Path, isWildcard)

			if !commonInterceptor.ProcessInterceptor(responseState, r, body, interceptorCfg, requestStore, h.imposterConfig, h.configDir, h) {
				responseState.Handled = true
				return // Short-circuit if interceptor continue is false
			}
		}
	}

	var matches []matcher.MatchResult
	for _, res := range h.config.Resources {
		score, isWildcard := matcher.CalculateMatchScore(&res.RequestMatcher, r, body)
		if score > 0 {
			matches = append(matches, matcher.MatchResult{Resource: &res, Score: score, Wildcard: isWildcard})
		}
	}

	if len(matches) == 0 {
		return // Let the main handler deal with no matches
	}

	// Find the best match
	best, tie := matcher.FindBestMatch(matches)
	if tie {
		fmt.Printf("Warning: multiple equally specific matches. Using the first.\n")
	}

	// Capture request data
	capture.CaptureRequestData(h.imposterConfig, *best.Resource, r, body, requestStore)

	// Process the response
	h.ProcessResponse(responseState, r, best.Resource.Response, requestStore)
	responseState.Handled = true
}

// ProcessResponse handles preparing the response state
func (h *Handler) ProcessResponse(rs *response.ResponseState, r *http.Request, resp config.Response, requestStore store.Store) {
	response.ProcessResponse(rs, r, resp, h.configDir, requestStore, h.imposterConfig)
}
