package rest

import (
	"net/http"

	"github.com/imposter-project/imposter-go/pkg/logger"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/common"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/steps"
)

// HandleRequest processes incoming REST API requests
func (h *PluginHandler) HandleRequest(
	exch *exchange.Exchange,
	respProc response.Processor,
) {
	r := exch.Request.Request
	responseState := exch.ResponseState

	// Get system XML namespaces
	var systemNamespaces map[string]string
	if h.config.System != nil {
		systemNamespaces = h.config.System.XMLNamespaces
	}

	// Process interceptors first
	for _, interceptorCfg := range h.config.Interceptors {
		score, _ := matcher.CalculateMatchScore(exch, &interceptorCfg.RequestMatcher, systemNamespaces, h.imposterConfig)
		if score > 0 {
			logger.Infof("matched interceptor - method:%s, path:%s", r.Method, r.URL.Path)
			if interceptorCfg.Capture != nil {
				capture.CaptureRequestData(h.imposterConfig, &interceptorCfg.RequestMatcher, interceptorCfg.Capture, exch)
			}

			// Execute steps if present
			if len(interceptorCfg.Steps) > 0 {
				if err := steps.RunSteps(interceptorCfg.Steps, exch, h.imposterConfig, h.configDir, responseState, &interceptorCfg.RequestMatcher); err != nil {
					logger.Errorf("failed to execute interceptor steps: %v", err)
					responseState.StatusCode = http.StatusInternalServerError
					responseState.Body = []byte("Failed to execute steps")
					responseState.Handled = true // Error case, no resource to attach
					return
				}
				if responseState.Handled {
					// Step(s) handled the request, so we don't need to process the response
					return
				}
			}

			if interceptorCfg.Response != nil {
				h.processResponse(exch, &interceptorCfg.RequestMatcher, interceptorCfg.Response, respProc)
			}
			if !interceptorCfg.Continue {
				responseState.HandledWithResource(&interceptorCfg.BaseResource)
				return // Short-circuit if interceptor continue is false
			}
		}
	}

	var matches []matcher.MatchResult
	for _, res := range h.config.Resources {
		score, isWildcard := matcher.CalculateMatchScore(exch, &res.RequestMatcher, systemNamespaces, h.imposterConfig)
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

	// Check rate limiting if configured
	if len(best.Resource.Concurrency) > 0 {
		processResponseFunc := func(exch *exchange.Exchange, requestMatcher *config.RequestMatcher, response *config.Response, respProc response.Processor) {
			h.processResponse(exch, requestMatcher, response, respProc)
		}

		shouldLimit, cleanupFunc := common.RateLimitCheck(
			best.Resource,
			best.Resource.Method,
			best.Resource.Path, // resourceName (path for REST)
			exch,
			respProc,
			processResponseFunc,
		)

		if shouldLimit {
			return
		}

		if cleanupFunc != nil {
			defer cleanupFunc()
		}
	}

	// Capture request data
	capture.CaptureRequestData(h.imposterConfig, &best.Resource.RequestMatcher, best.Resource.Capture, exch)

	// Execute steps if present
	if len(best.Resource.Steps) > 0 {
		if err := steps.RunSteps(best.Resource.Steps, exch, h.imposterConfig, h.configDir, responseState, &best.Resource.RequestMatcher); err != nil {
			logger.Errorf("failed to execute resource steps: %v", err)
			responseState.StatusCode = http.StatusInternalServerError
			responseState.Body = []byte("Failed to execute steps")
			responseState.Handled = true // Error case, no resource to attach
			return
		}
		if responseState.Handled {
			// Step(s) handled the request, so we don't need to process the response
			return
		}
	}

	// Process the response
	if best.Resource.Response != nil {
		h.processResponse(exch, &best.Resource.RequestMatcher, best.Resource.Response, respProc)
		responseState.HandledWithResource(&best.Resource.BaseResource)
	}
}
