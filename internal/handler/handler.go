package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/plugin"
	"github.com/imposter-project/imposter-go/internal/soap"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/internal/template"
)

// matchResult represents a match between a request and a resource or interceptor
type matchResult struct {
	Resource    *config.Resource
	Interceptor *config.Interceptor
	Score       int
	Wildcard    bool
}

// responseState tracks the state of the HTTP response
type responseState struct {
	statusCode int
	headers    map[string]string
	body       []byte
	completed  bool // indicates if the response is complete (e.g., connection closed)
}

// newResponseState creates a new responseState with default values
func newResponseState() *responseState {
	return &responseState{
		statusCode: http.StatusOK,
		headers:    make(map[string]string),
	}
}

// writeToResponseWriter writes the final state to the http.ResponseWriter
func (rs *responseState) writeToResponseWriter(w http.ResponseWriter) {
	if rs.completed {
		// Handle connection closing
		if hijacker, ok := w.(http.Hijacker); ok {
			if conn, _, err := hijacker.Hijack(); err == nil {
				conn.Close()
				return
			}
		}
		// Fallback if hijacking is not supported
		rs.statusCode = http.StatusInternalServerError
		rs.body = []byte("HTTP server does not support connection hijacking")
	}

	for key, value := range rs.headers {
		w.Header().Set(key, value)
	}
	w.WriteHeader(rs.statusCode)
	if rs.body != nil {
		w.Write(rs.body)
	}
}

// HandleRequest processes incoming HTTP requests and routes them to the appropriate handler
func HandleRequest(w http.ResponseWriter, r *http.Request, configDir string, configs []config.Config, imposterConfig *config.ImposterConfig) {
	// Handle system endpoints
	if handleSystemEndpoint(w, r) {
		return
	}

	// Process each config
	for _, cfg := range configs {
		switch cfg.Plugin {
		case "rest":
			handleRestRequest(w, r, &cfg, configDir, imposterConfig)
		case "soap":
			handleSOAPRequest(w, r, &cfg, configDir)
		default:
			http.Error(w, "Unsupported plugin type", http.StatusInternalServerError)
			return
		}
	}
}

// handleSystemEndpoint handles system-level endpoints like /system/store
func handleSystemEndpoint(w http.ResponseWriter, r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, "/system/store") {
		HandleStoreRequest(w, r)
		return true
	}
	return false
}

// handleRestRequest handles REST API requests
func handleRestRequest(w http.ResponseWriter, r *http.Request, cfg *config.Config, configDir string, imposterConfig *config.ImposterConfig) {
	body, err := plugin.GetRequestBody(r)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Initialize request-scoped store
	requestStore := make(store.Store)
	responseState := newResponseState()

	// Process interceptors first
	for _, interceptor := range cfg.Interceptors {
		score, isWildcard := plugin.CalculateMatchScore(&interceptor.RequestMatcher, r, body)
		if score > 0 {
			fmt.Printf("Matched interceptor - method:%s, path:%s, wildcard:%v\n",
				r.Method, r.URL.Path, isWildcard)
			// Process the interceptor
			if !processInterceptor(responseState, r, body, interceptor, configDir, imposterConfig, requestStore) {
				responseState.writeToResponseWriter(w)
				return // Short-circuit if interceptor responded and continue is false
			}
		}
	}

	var matches []plugin.MatchResult
	for _, res := range cfg.Resources {
		score, isWildcard := plugin.CalculateMatchScore(&res.RequestMatcher, r, body)
		if score > 0 {
			matches = append(matches, plugin.MatchResult{Resource: &res, Score: score, Wildcard: isWildcard})
		}
	}

	if len(matches) == 0 {
		notFoundMsg := "Resource not found"
		responseState.statusCode = http.StatusNotFound
		responseState.body = []byte(notFoundMsg)
		responseState.writeToResponseWriter(w)
		fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
			r.Method, r.URL.Path, http.StatusNotFound, len(notFoundMsg))
		return
	}

	// Find the best match
	best, tie := plugin.FindBestMatch(matches)
	if tie {
		fmt.Printf("Warning: multiple equally specific matches. Using the first.\n")
	}

	// Capture request data
	capture.CaptureRequestData(imposterConfig, *best.Resource, r, body, requestStore)

	// Process the response
	processResponse(responseState, r, best.Resource.Response, configDir, imposterConfig, requestStore)
	responseState.writeToResponseWriter(w)
}

// handleSOAPRequest handles SOAP requests using the SOAP plugin
func handleSOAPRequest(w http.ResponseWriter, r *http.Request, cfg *config.Config, configDir string) {
	handler, err := soap.NewHandler(cfg, configDir)
	if err != nil {
		http.Error(w, "Failed to initialize SOAP handler", http.StatusInternalServerError)
		return
	}
	handler.HandleRequest(w, r)
}

// processInterceptor handles an interceptor and returns true if request processing should continue
func processInterceptor(rs *responseState, r *http.Request, body []byte, interceptor config.Interceptor, configDir string, imposterConfig *config.ImposterConfig, requestStore store.Store) bool {
	// Capture request data if specified
	if interceptor.Capture != nil {
		capture.CaptureRequestData(imposterConfig, config.Resource{
			RequestMatcher: config.RequestMatcher{
				Capture: interceptor.Capture,
			},
		}, r, body, requestStore)
	}

	// If the interceptor has a response and continue is false, send the response and stop processing
	if interceptor.Response != nil {
		processResponse(rs, r, *interceptor.Response, configDir, imposterConfig, requestStore)
	}

	return interceptor.Continue
}

// processResponse handles preparing the response state
func processResponse(rs *responseState, r *http.Request, response config.Response, configDir string, imposterConfig *config.ImposterConfig, requestStore store.Store) {
	// Handle delay if specified
	if response.Delay.Exact > 0 {
		delay := response.Delay.Exact
		fmt.Printf("Delaying request (exact: %dms) - method:%s, path:%s\n", delay, r.Method, r.URL.Path)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	} else if response.Delay.Min > 0 && response.Delay.Max > 0 {
		delay := rand.Intn(response.Delay.Max-response.Delay.Min+1) + response.Delay.Min
		fmt.Printf("Delaying request (range: %dms-%dms, actual: %dms) - method:%s, path:%s\n",
			response.Delay.Min, response.Delay.Max, delay, r.Method, r.URL.Path)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	if response.StatusCode > 0 {
		rs.statusCode = response.StatusCode
	}

	// Set response headers
	for key, value := range response.Headers {
		rs.headers[key] = value
	}

	var responseContent string
	if response.File != "" {
		filePath := filepath.Join(configDir, response.File)
		data, err := os.ReadFile(filePath)
		if err != nil {
			rs.statusCode = http.StatusInternalServerError
			rs.body = []byte("Failed to read file")
			return
		}
		responseContent = string(data)
	} else {
		responseContent = response.Content
	}

	if response.Template {
		responseContent = template.ProcessTemplate(responseContent, r, imposterConfig, requestStore)
	}

	if response.Fail != "" {
		switch response.Fail {
		case "EmptyResponse":
			// Send a status but no body
			rs.body = nil
			fmt.Printf("Handled request (simulated failure: EmptyResponse) - method:%s, path:%s, status:%d, length:0\n",
				r.Method, r.URL.Path, rs.statusCode)
			return

		case "CloseConnection":
			// Mark the response as completed to prevent writing the body
			rs.completed = true
			fmt.Printf("Handled request (simulated failure: CloseConnection) - method:%s, path:%s\n", r.Method, r.URL.Path)
			return
		}
	}

	rs.body = []byte(responseContent)
	fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
		r.Method, r.URL.Path, rs.statusCode, len(responseContent))
}

// matchBodyCondition handles matching a single body condition against the request body
func matchBodyCondition(body []byte, condition config.BodyMatchCondition) bool {
	if condition.JSONPath != "" {
		return matcher.MatchJSONPath(body, condition)
	} else if condition.XPath != "" {
		return matcher.MatchXPath(body, condition)
	}
	return condition.Match(string(body))
}

// calculateMatchScore calculates how well a request matches a resource or interceptor
func calculateMatchScore(matcher interface{}, r *http.Request, body []byte) (int, bool) {
	var requestMatcher *config.RequestMatcher

	switch m := matcher.(type) {
	case *config.Resource:
		requestMatcher = &m.RequestMatcher
	case *config.Interceptor:
		requestMatcher = &m.RequestMatcher
	default:
		return 0, false
	}

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
		if !matchBodyCondition(body, requestMatcher.RequestBody.BodyMatchCondition) {
			return 0, false
		}
		score++
	} else if len(requestMatcher.RequestBody.AllOf) > 0 {
		for _, condition := range requestMatcher.RequestBody.AllOf {
			if !matchBodyCondition(body, condition) {
				return 0, false
			}
		}
		score++
	} else if len(requestMatcher.RequestBody.AnyOf) > 0 {
		matched := false
		for _, condition := range requestMatcher.RequestBody.AnyOf {
			if matchBodyCondition(body, condition) {
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
