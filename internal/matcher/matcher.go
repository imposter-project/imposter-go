package matcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/imposter-project/imposter-go/internal/config"
)

// MatchResult represents a match between a request and a resource or interceptor
type MatchResult struct {
	Resource    *config.Resource
	Interceptor *config.Interceptor
	Score       int
	Wildcard    bool
}

// CalculateMatchScore calculates how well a request matches a resource or interceptor
func CalculateMatchScore(matcher *config.RequestMatcher, r *http.Request, body []byte) (score int, isWildcard bool) {
	// Method match
	if matcher.Method != "" {
		if matcher.Method != r.Method {
			return 0, false
		}
		score++
	}

	// Path match
	if matcher.Path != "" {
		// Split paths into segments
		resourceSegments := strings.Split(strings.Trim(matcher.Path, "/"), "/")
		requestSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

		// Check for trailing wildcard
		if len(resourceSegments) > 0 && resourceSegments[len(resourceSegments)-1] == "*" {
			isWildcard = true
			resourceSegments = resourceSegments[:len(resourceSegments)-1]
			// For wildcard matches, we require at least the base path to match
			if len(requestSegments) < len(resourceSegments) {
				return 0, false
			}
			requestSegments = requestSegments[:len(resourceSegments)]
		} else if len(resourceSegments) != len(requestSegments) {
			return 0, false
		}

		// Match path segments, including path parameters
		for i, segment := range resourceSegments {
			if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
				paramName := strings.Trim(segment, "{}")
				if condition, hasParam := matcher.PathParams[paramName]; hasParam {
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
	}

	// Headers match
	for key, condition := range matcher.Headers {
		actualValue := r.Header.Get(key)
		if !condition.Matcher.Match(actualValue) {
			return 0, false
		}
		score++
	}

	// Query params match
	for key, condition := range matcher.QueryParams {
		actualValue := r.URL.Query().Get(key)
		if !condition.Matcher.Match(actualValue) {
			return 0, false
		}
		score++
	}

	// Form params match
	if len(matcher.FormParams) > 0 {
		if err := r.ParseForm(); err != nil {
			return 0, false
		}
		for key, condition := range matcher.FormParams {
			actualValue := r.FormValue(key)
			if !condition.Matcher.Match(actualValue) {
				return 0, false
			}
			score++
		}
	}

	// Request body match
	if matcher.RequestBody.Value != "" || matcher.RequestBody.JSONPath != "" || matcher.RequestBody.XPath != "" {
		if !matchBodyCondition(body, matcher.RequestBody.BodyMatchCondition) {
			return 0, false
		}
		score++
	} else if len(matcher.RequestBody.AllOf) > 0 {
		for _, condition := range matcher.RequestBody.AllOf {
			if !matchBodyCondition(body, condition) {
				return 0, false
			}
		}
		score++
	} else if len(matcher.RequestBody.AnyOf) > 0 {
		matched := false
		for _, condition := range matcher.RequestBody.AnyOf {
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

	return score, isWildcard
}

// matchBodyCondition checks if a body condition matches the request body
func matchBodyCondition(body []byte, condition config.BodyMatchCondition) bool {
	if condition.JSONPath != "" {
		return MatchJSONPath(body, condition)
	} else if condition.XPath != "" {
		return MatchXPath(body, condition)
	}
	return condition.Match(string(body))
}

// FindBestMatch finds the best matching resource from a list of matches
func FindBestMatch(matches []MatchResult) (best MatchResult, tie bool) {
	if len(matches) == 0 {
		return MatchResult{}, false
	}

	best = matches[0]
	tie = false

	for _, m := range matches[1:] {
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

	return best, tie
}

// GetRequestBody reads and resets the request body
func GetRequestBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// MatchXPath matches XML body content using XPath query
func MatchXPath(body []byte, condition config.BodyMatchCondition) bool {
	doc, err := xmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		return false
	}

	// Compile an XPath expression with namespace bindings.
	// The map keys are the prefixes (e.g. "ns1"), and the values are the namespace URIs.
	expr, err := xpath.CompileWithNS(
		condition.XPath,
		condition.XMLNamespaces,
	)
	if err != nil {
		panic(err)
	}

	// Select the node using the compiled expression.
	result := xmlquery.QuerySelector(doc, expr)
	if result == nil {
		return condition.Match("")
	}

	return condition.Match(result.InnerText())
}

// MatchJSONPath matches JSON body content using JSONPath query
func MatchJSONPath(body []byte, condition config.BodyMatchCondition) bool {
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return false
	}

	results, err := jsonpath.Get(condition.JSONPath, jsonData)
	if err != nil {
		return false
	}

	return condition.Match(results.(string))
}
