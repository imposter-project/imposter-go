package handler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/template"
	"golang.org/x/exp/rand"
)

// matchResult represents a match between a request and a resource
type matchResult struct {
	Resource config.Resource
	Score    int
	Wildcard bool
}

// HandleRequest processes incoming HTTP requests based on resources
func HandleRequest(w http.ResponseWriter, r *http.Request, configDir string, configs []config.Config, imposterConfig *config.ImposterConfig) {
	body, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewReader(body))

	var allMatches []matchResult

	for _, cfg := range configs {
		for _, res := range cfg.Resources {
			score, isWildcard := calculateMatchScore(res, r, body)
			if score > 0 {
				allMatches = append(allMatches, matchResult{Resource: res, Score: score, Wildcard: isWildcard})
			}
		}
	}

	if len(allMatches) == 0 {
		notFoundMsg := "Resource not found"
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, notFoundMsg)
		fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
			r.Method, r.URL.Path, http.StatusNotFound, len(notFoundMsg))
		return
	}

	// Find the best match considering both score and wildcard status
	best := allMatches[0]
	tie := false
	for _, m := range allMatches[1:] {
		// If scores are equal, prefer non-wildcard matches
		if m.Score == best.Score {
			if best.Wildcard && !m.Wildcard {
				best = m
				tie = false
			} else if !best.Wildcard && !m.Wildcard || best.Wildcard && m.Wildcard {
				tie = true
			}
		} else if m.Score > best.Score {
			best = m
			tie = false
		}
	}

	if tie {
		fmt.Printf("Warning: multiple equally specific matches. Using the first.\n")
	}

	// Initialize request-scoped store
	requestStore := make(map[string]interface{})

	// Capture request data
	capture.CaptureRequestData(imposterConfig, best.Resource, r, body, requestStore)

	// Handle delay if specified
	if best.Resource.Response.Delay.Exact > 0 {
		delay := best.Resource.Response.Delay.Exact
		fmt.Printf("Delaying request (exact: %dms) - method:%s, path:%s\n", delay, r.Method, r.URL.Path)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	} else if best.Resource.Response.Delay.Min > 0 && best.Resource.Response.Delay.Max > 0 {
		delay := rand.Intn(best.Resource.Response.Delay.Max-best.Resource.Response.Delay.Min+1) + best.Resource.Response.Delay.Min
		fmt.Printf("Delaying request (range: %dms-%dms, actual: %dms) - method:%s, path:%s\n",
			best.Resource.Response.Delay.Min, best.Resource.Response.Delay.Max, delay, r.Method, r.URL.Path)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	// Write response using 'best.Resource'
	statusCode := best.Resource.Response.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}

	// Set response headers
	for key, value := range best.Resource.Response.Headers {
		w.Header().Set(key, value)
	}

	var responseContent string
	if best.Resource.Response.File != "" {
		filePath := filepath.Join(configDir, best.Resource.Response.File)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
		responseContent = string(data)
	} else {
		responseContent = best.Resource.Response.Content
	}

	if best.Resource.Response.Template {
		responseContent = template.ProcessTemplate(responseContent, r, imposterConfig, requestStore)
	}

	if best.Resource.Response.Fail != "" {
		switch best.Resource.Response.Fail {
		case "EmptyResponse":
			// Send a status but no body
			w.WriteHeader(statusCode)
			fmt.Printf("Handled request (simulated failure: EmptyResponse) - method:%s, path:%s, status:%d, length:0\n",
				r.Method, r.URL.Path, statusCode)
			return

		case "CloseConnection":
			// Close the connection before sending any response
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "HTTP server does not support connection hijacking", http.StatusInternalServerError)
				return
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
				return
			}
			fmt.Printf("Handled request (simulated failure: CloseConnection) - method:%s, path:%s\n", r.Method, r.URL.Path)
			conn.Close()
			return
		}
	}

	w.Write([]byte(responseContent))
	fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
		r.Method, r.URL.Path, statusCode, len(responseContent))
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

// calculateMatchScore returns the match score and whether the match used a wildcard.
// Returns (0, false) if any required condition fails, meaning no match.
func calculateMatchScore(res config.Resource, r *http.Request, body []byte) (int, bool) {
	score := 0

	// Match method
	if r.Method != res.Method {
		return 0, false
	}
	score++

	// Match path with optional pathParams and trailing wildcard
	requestSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	resourceSegments := strings.Split(strings.Trim(res.Path, "/"), "/")

	// Check for trailing wildcard
	hasWildcard := len(resourceSegments) > 0 && resourceSegments[len(resourceSegments)-1] == "*"
	if hasWildcard {
		resourceSegments = resourceSegments[:len(resourceSegments)-1]
		// For wildcard matches, we require at least the base path to match
		if len(requestSegments) < len(resourceSegments) {
			return 0, false
		}
	} else if len(requestSegments) != len(resourceSegments) {
		return 0, false
	}

	// Match path segments
	for i, segment := range resourceSegments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			paramName := strings.Trim(segment, "{}")
			if condition, hasParam := res.PathParams[paramName]; hasParam {
				if !condition.Matcher.Match(requestSegments[i]) {
					return 0, false
				}
				score++
			}
		} else {
			if requestSegments[i] != segment {
				return 0, false
			}
		}
	}

	// Match query parameters
	for key, condition := range res.QueryParams {
		actualValue := r.URL.Query().Get(key)
		if !condition.Matcher.Match(actualValue) {
			return 0, false
		}
		score++
	}

	// Match headers
	for key, condition := range res.Headers {
		actualValue := r.Header.Get(key)
		if !condition.Matcher.Match(actualValue) {
			return 0, false
		}
		score++
	}

	// Match form parameters (if content type is application/x-www-form-urlencoded)
	if len(res.FormParams) > 0 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return 0, false
		}
		for key, condition := range res.FormParams {
			if !condition.Matcher.Match(r.FormValue(key)) {
				return 0, false
			}
			score++
		}
	}

	// Match request body
	if res.RequestBody.JSONPath != "" || res.RequestBody.XPath != "" || res.RequestBody.Value != "" {
		if !matchBodyCondition(body, res.RequestBody.BodyMatchCondition) {
			return 0, false
		}
		score++
	} else if len(res.RequestBody.AllOf) > 0 {
		for _, condition := range res.RequestBody.AllOf {
			if !matchBodyCondition(body, condition) {
				return 0, false
			}
		}
		score++
	} else if len(res.RequestBody.AnyOf) > 0 {
		matched := false
		for _, condition := range res.RequestBody.AnyOf {
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
