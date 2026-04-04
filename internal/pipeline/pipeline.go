package pipeline

import (
	"net/http"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/common"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/steps"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// ScoreCalculator computes a match score for a request against a matcher.
// Returns score (negative means no match) and whether the match is wildcard.
type ScoreCalculator func(
	exch *exchange.Exchange,
	reqMatcher *config.RequestMatcher,
	systemNamespaces map[string]string,
	imposterConfig *config.ImposterConfig,
) (score int, isWildcard bool)

// StepErrorHandler formats a protocol-specific error onto responseState
// when step execution fails.
type StepErrorHandler func(responseState *exchange.ResponseState, msg string)

// ResponseHandler wraps or replaces the standard response processing with
// protocol-specific logic (e.g. SOAP envelope wrapping).
type ResponseHandler func(
	exch *exchange.Exchange,
	reqMatcher *config.RequestMatcher,
	resp *config.Response,
	respProc response.Processor,
)

// ResourceNamer extracts the name and method used for rate-limit key generation.
type ResourceNamer func(resource *config.Resource) (name string, method string)

// ProtocolHooks allows protocol-specific plugins to customise each
// extension point of the shared pipeline. Nil fields use defaults.
type ProtocolHooks struct {
	CalculateScore  ScoreCalculator
	OnStepError     StepErrorHandler
	ProcessResponse ResponseHandler
	GetResourceName ResourceNamer
}

// RunPipeline executes the common request processing pipeline:
//  1. Process interceptors (match, capture, steps, response)
//  2. Match all resources and find the best match
//  3. Rate-limit check
//  4. Capture request data
//  5. Execute steps
//  6. Process response
func RunPipeline(
	cfg *config.Config,
	imposterConfig *config.ImposterConfig,
	exch *exchange.Exchange,
	respProc response.Processor,
	hooks *ProtocolHooks,
) {
	if hooks == nil {
		hooks = &ProtocolHooks{}
	}

	r := exch.Request.Request
	responseState := exch.ResponseState

	// Resolve system XML namespaces
	var systemNamespaces map[string]string
	if cfg.System != nil {
		systemNamespaces = cfg.System.XMLNamespaces
	}

	calcScore := hooks.CalculateScore
	if calcScore == nil {
		calcScore = matcher.CalculateMatchScore
	}

	onStepError := hooks.OnStepError
	if onStepError == nil {
		onStepError = defaultStepErrorHandler
	}

	processResp := hooks.ProcessResponse
	if processResp == nil {
		processResp = defaultProcessResponse
	}

	// Process interceptors first
	for _, interceptorCfg := range cfg.Interceptors {
		score, _ := calcScore(exch, &interceptorCfg.RequestMatcher, systemNamespaces, imposterConfig)
		if score > 0 {
			logger.Infof("matched interceptor - method:%s, path:%s", r.Method, r.URL.Path)
			if interceptorCfg.Capture != nil {
				capture.CaptureRequestData(imposterConfig, &interceptorCfg.RequestMatcher, interceptorCfg.Capture, exch)
			}

			// Execute steps if present
			if len(interceptorCfg.Steps) > 0 {
				if err := steps.RunSteps(interceptorCfg.Steps, exch, imposterConfig, cfg.ConfigDir, responseState, &interceptorCfg.RequestMatcher); err != nil {
					logger.Errorf("failed to execute interceptor steps: %v", err)
					onStepError(responseState, "Failed to execute steps")
					return
				}
				if responseState.Handled {
					return
				}
			}

			if interceptorCfg.Response != nil {
				processResp(exch, &interceptorCfg.RequestMatcher, interceptorCfg.Response, respProc)
			}
			if !interceptorCfg.Continue {
				responseState.HandledWithResource(&interceptorCfg.BaseResource)
				return
			}
		}
	}

	// Match all resources
	var matches []matcher.MatchResult
	for _, res := range cfg.Resources {
		score, isWildcard := calcScore(exch, &res.RequestMatcher, systemNamespaces, imposterConfig)
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
		resourceName, resourceMethod := getResourceName(hooks, best.Resource)

		shouldLimit := common.RateLimitCheck(
			best.Resource,
			resourceMethod,
			resourceName,
			exch,
			respProc,
			processResp,
		)

		if shouldLimit {
			return
		}
	}

	// Capture request data
	capture.CaptureRequestData(imposterConfig, &best.Resource.RequestMatcher, best.Resource.Capture, exch)

	// Execute steps if present
	if len(best.Resource.Steps) > 0 {
		if err := steps.RunSteps(best.Resource.Steps, exch, imposterConfig, cfg.ConfigDir, responseState, &best.Resource.RequestMatcher); err != nil {
			logger.Errorf("failed to execute resource steps: %v", err)
			onStepError(responseState, "Failed to execute steps")
			return
		}
		if responseState.Handled {
			return
		}
	}

	// Process the response if a response block is configured
	if best.Resource.Response != nil {
		processResp(exch, &best.Resource.RequestMatcher, best.Resource.Response, respProc)
	}

	// If we matched a resource, ensure the request is marked as handled
	// even if there's no response block (e.g. steps modified response
	// directly or only set a status code).
	if !responseState.Handled {
		responseState.HandledWithResource(&best.Resource.BaseResource)
	}
}

func defaultStepErrorHandler(responseState *exchange.ResponseState, msg string) {
	responseState.StatusCode = http.StatusInternalServerError
	responseState.Body = []byte(msg)
	responseState.Handled = true
}

func defaultProcessResponse(
	exch *exchange.Exchange,
	reqMatcher *config.RequestMatcher,
	resp *config.Response,
	respProc response.Processor,
) {
	respProc(exch, reqMatcher, resp)
}

func getResourceName(hooks *ProtocolHooks, resource *config.Resource) (string, string) {
	if hooks.GetResourceName != nil {
		return hooks.GetResourceName(resource)
	}
	return resource.Path, resource.Method
}
