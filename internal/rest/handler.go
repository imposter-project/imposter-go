package rest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/matcher"
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
func (h *Handler) HandleRequest(r *http.Request) *plugin.ResponseState {
	body, err := plugin.GetRequestBody(r)
	if err != nil {
		rs := plugin.NewResponseState()
		rs.StatusCode = http.StatusBadRequest
		rs.Body = []byte("Failed to read request body")
		return rs
	}

	// Initialize request-scoped store
	requestStore := make(store.Store)
	responseState := plugin.NewResponseState()

	// Process interceptors first
	for _, interceptor := range h.config.Interceptors {
		score, isWildcard := h.calculateMatchScore(&interceptor.RequestMatcher, r, body)
		if score > 0 {
			fmt.Printf("Matched interceptor - method:%s, path:%s, wildcard:%v\n",
				r.Method, r.URL.Path, isWildcard)
			// Process the interceptor
			if !h.processInterceptor(responseState, r, body, interceptor, requestStore) {
				return responseState // Short-circuit if interceptor responded and continue is false
			}
		}
	}

	var matches []plugin.MatchResult
	for _, res := range h.config.Resources {
		score, isWildcard := h.calculateMatchScore(&res.RequestMatcher, r, body)
		if score > 0 {
			matches = append(matches, plugin.MatchResult{Resource: &res, Score: score, Wildcard: isWildcard})
		}
	}

	if len(matches) == 0 {
		notFoundMsg := "Resource not found"
		responseState.StatusCode = http.StatusNotFound
		responseState.Body = []byte(notFoundMsg)
		fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
			r.Method, r.URL.Path, http.StatusNotFound, len(notFoundMsg))
		return responseState
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
	return responseState
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

// matchBodyCondition handles matching a single body condition against the request body
func (h *Handler) matchBodyCondition(body []byte, condition config.BodyMatchCondition) bool {
	if condition.JSONPath != "" {
		return matcher.MatchJSONPath(body, condition)
	} else if condition.XPath != "" {
		return matcher.MatchXPath(body, condition)
	}
	return condition.Match(string(body))
}

// calculateMatchScore calculates how well a request matches a resource or interceptor
func (h *Handler) calculateMatchScore(requestMatcher *config.RequestMatcher, r *http.Request, body []byte) (int, bool) {
	score := 0
	hasWildcard := false

	// Match HTTP method
	if requestMatcher.Method != "" && requestMatcher.Method != r.Method {
		return 0, false
	}
	score++

	// Match path
	if requestMatcher.Path == "" {
		return 0, false
	}

	// Split paths into segments
	resourceSegments := strings.Split(strings.Trim(requestMatcher.Path, "/"), "/")
	requestSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	// Check for trailing wildcard
	if len(resourceSegments) > 0 && resourceSegments[len(resourceSegments)-1] == "*" {
		hasWildcard = true
		resourceSegments = resourceSegments[:len(resourceSegments)-1]
		// For wildcard matches, we require at least the base path to match
		if len(requestSegments) < len(resourceSegments) {
			return 0, false
		}
		requestSegments = requestSegments[:len(resourceSegments)]
	} else if len(resourceSegments) != len(requestSegments) {
		return 0, false
	}

	// Match path segments
	for i, segment := range resourceSegments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			paramName := strings.Trim(segment, "{}")
			if condition, hasParam := requestMatcher.PathParams[paramName]; hasParam {
				if !condition.Matcher.Match(requestSegments[i]) {
					return 0, false
				}
				score++
			}
		} else {
			if requestSegments[i] != segment {
				return 0, false
			}
			score++
		}
	}

	// Match query parameters
	for key, condition := range requestMatcher.QueryParams {
		actualValue := r.URL.Query().Get(key)
		if !condition.Matcher.Match(actualValue) {
			return 0, false
		}
		score++
	}

	// Match headers
	for key, condition := range requestMatcher.Headers {
		actualValue := r.Header.Get(key)
		if !condition.Matcher.Match(actualValue) {
			return 0, false
		}
		score++
	}

	// Match form parameters (if content type is application/x-www-form-urlencoded)
	if len(requestMatcher.FormParams) > 0 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return 0, false
		}
		for key, condition := range requestMatcher.FormParams {
			if !condition.Matcher.Match(r.FormValue(key)) {
				return 0, false
			}
			score++
		}
	}

	// Match request body
	if requestMatcher.RequestBody.JSONPath != "" || requestMatcher.RequestBody.XPath != "" || requestMatcher.RequestBody.Value != "" {
		if !h.matchBodyCondition(body, requestMatcher.RequestBody.BodyMatchCondition) {
			return 0, false
		}
		score++
	} else if len(requestMatcher.RequestBody.AllOf) > 0 {
		for _, condition := range requestMatcher.RequestBody.AllOf {
			if !h.matchBodyCondition(body, condition) {
				return 0, false
			}
		}
		score++
	} else if len(requestMatcher.RequestBody.AnyOf) > 0 {
		matched := false
		for _, condition := range requestMatcher.RequestBody.AnyOf {
			if h.matchBodyCondition(body, condition) {
				matched = true
				break
			}
		}
		if !matched {
			return 0, false
		}
		score++
	}

	return score, hasWildcard
}
