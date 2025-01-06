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
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/internal/template"
)

// MatchResult represents a match between a request and a resource or interceptor
type MatchResult struct {
	Resource    *config.Resource
	Interceptor *config.Interceptor
	Score       int
	Wildcard    bool
}

// CalculateMatchScore calculates how well a request matches a resource or interceptor
func CalculateMatchScore(matcher *config.RequestMatcher, r *http.Request, body []byte, systemNamespaces map[string]string, imposterConfig *config.ImposterConfig, requestStore store.Store) (score int, isWildcard bool) {
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
	for key, condition := range matcher.RequestHeaders {
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
		if !matchBodyCondition(body, matcher.RequestBody.BodyMatchCondition, systemNamespaces) {
			return 0, false
		}
		score++
	} else if len(matcher.RequestBody.AllOf) > 0 {
		for _, condition := range matcher.RequestBody.AllOf {
			if !matchBodyCondition(body, condition, systemNamespaces) {
				return 0, false
			}
		}
		score += len(matcher.RequestBody.AllOf)
	} else if len(matcher.RequestBody.AnyOf) > 0 {
		matched := false
		for _, condition := range matcher.RequestBody.AnyOf {
			if matchBodyCondition(body, condition, systemNamespaces) {
				matched = true
				break
			}
		}
		if !matched {
			return 0, false
		}
		score++
	}

	// All expressions must match
	if len(matcher.AllOf) > 0 {
		for _, expr := range matcher.AllOf {
			// Evaluate the expression using the template engine
			result, err := evaluateExpression(expr.Expression, r, imposterConfig, requestStore)
			if err != nil {
				return 0, false
			}
			if !expr.MatchCondition.Match(result) {
				return 0, false
			}
		}
		score += len(matcher.AllOf)

		// At least one expression must match
	} else if len(matcher.AnyOf) > 0 {
		matched := false
		for _, expr := range matcher.AnyOf {
			// Evaluate the expression using the template engine
			result, err := evaluateExpression(expr.Expression, r, imposterConfig, requestStore)
			if err != nil {
				continue
			}
			if expr.MatchCondition.Match(result) {
				matched = true
				break
			}
		}
		if !matched {
			return 0, false
		}
		score++
	}

	// Return the score and wildcard status for path-based matches
	return score, isWildcard
}

// evaluateExpression evaluates a template expression in the context of the request
func evaluateExpression(expression string, r *http.Request, imposterConfig *config.ImposterConfig, requestStore store.Store) (string, error) {
	// Simply evaluate the expression and return its value
	return template.ProcessTemplate(expression, r, imposterConfig, requestStore), nil
}

// matchBodyCondition checks if a body condition matches the request body
func matchBodyCondition(body []byte, condition config.BodyMatchCondition, systemNamespaces map[string]string) bool {
	if condition.JSONPath != "" {
		return MatchJSONPath(body, condition)
	} else if condition.XPath != "" {
		return MatchXPath(body, condition, systemNamespaces)
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
func MatchXPath(body []byte, condition config.BodyMatchCondition, systemNamespaces map[string]string) bool {
	doc, err := xmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		return false
	}

	// Merge system namespaces with condition namespaces, giving precedence to condition namespaces
	namespaces := make(map[string]string)
	for k, v := range systemNamespaces {
		namespaces[k] = v
	}
	for k, v := range condition.XMLNamespaces {
		namespaces[k] = v
	}

	// Compile an XPath expression with namespace bindings.
	// The map keys are the prefixes (e.g. "ns1"), and the values are the namespace URIs.
	expr, err := xpath.CompileWithNS(
		condition.XPath,
		namespaces,
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

	// Handle different result types
	switch v := results.(type) {
	case string:
		return condition.Match(v)
	case []interface{}:
		// For array results, check if any element matches
		for _, item := range v {
			if str, ok := item.(string); ok {
				if condition.Match(str) {
					return true
				}
			}
		}
		return false
	case float64:
		return condition.Match(fmt.Sprintf("%v", v))
	case bool:
		return condition.Match(fmt.Sprintf("%v", v))
	case nil:
		return condition.Match("")
	default:
		// Try to convert to string as a last resort
		return condition.Match(fmt.Sprintf("%v", v))
	}
}
