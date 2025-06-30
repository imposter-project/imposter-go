package matcher

import (
	"bytes"
	"fmt"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/query"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/template"
)

// MatchResult represents a match between a request and a resource or interceptor
type MatchResult struct {
	Resource    *config.Resource
	Interceptor *config.Interceptor
	Score       int
	Wildcard    bool

	// Whether the match was from a runtime-generated resource or interceptor
	RuntimeGenerated bool
}

const (
	NegativeMatchScore = -1
)

// CalculateMatchScore calculates how well a request matches a resource or interceptor.
// The score is calculated based on the number of matching conditions.
// If score is negative, the request explicitly does not match the resource.
// If the score is zero, no conditions were specified by the resource matcher.
func CalculateMatchScore(exch *exchange.Exchange, matcher *config.RequestMatcher, systemNamespaces map[string]string, imposterConfig *config.ImposterConfig) (score int, isWildcard bool) {
	r := exch.Request.Request

	// Method match
	if matcher.Method != "" {
		if !strings.EqualFold(matcher.Method, r.Method) {
			return NegativeMatchScore, false
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
				return NegativeMatchScore, false
			}
			requestSegments = requestSegments[:len(resourceSegments)]
		} else if len(resourceSegments) != len(requestSegments) {
			return NegativeMatchScore, false
		}

		// Match path segments, including path parameters
		for i, segment := range resourceSegments {
			if strings.Contains(segment, "{") && strings.Contains(segment, "}") {
				// Check if it's a pure parameter (entire segment is just {param}) or mixed
				if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") && strings.Count(segment, "{") == 1 && strings.Count(segment, "}") == 1 {
					// Pure parameter segment like {id}
					paramName := strings.Trim(segment, "{}")
					if condition, hasParam := matcher.PathParams[paramName]; hasParam {
						if !condition.Matcher.Match(requestSegments[i]) {
							return NegativeMatchScore, false
						}
						score++
					}
				} else {
					// Mixed segment with parameters and literal text like {param}.diff or {name}.{ext}
					matched, segmentScore := matchMixedSegment(segment, requestSegments[i], matcher.PathParams)
					if !matched {
						return NegativeMatchScore, false
					}
					score += segmentScore
				}
			} else {
				// Exact literal segment
				if requestSegments[i] != segment {
					return NegativeMatchScore, false
				}
				score++
			}
		}
	}

	// Headers match
	for key, condition := range matcher.RequestHeaders {
		actualValue := r.Header.Get(key)
		if !condition.Matcher.Match(actualValue) {
			return NegativeMatchScore, false
		}
		score++
	}

	// Query params match
	for key, condition := range matcher.QueryParams {
		actualValue := r.URL.Query().Get(key)
		if !condition.Matcher.Match(actualValue) {
			return NegativeMatchScore, false
		}
		score++
	}

	// Form params match
	if len(matcher.FormParams) > 0 {
		if err := r.ParseForm(); err != nil {
			return NegativeMatchScore, false
		}
		for key, condition := range matcher.FormParams {
			actualValue := r.FormValue(key)
			if !condition.Matcher.Match(actualValue) {
				return NegativeMatchScore, false
			}
			score++
		}
	}

	// Request body match
	if hasSingleBodyMatcher(matcher) {
		if !matchBodyCondition(exch.Request.Body, *matcher.RequestBody.BodyMatchCondition, systemNamespaces) {
			return NegativeMatchScore, false
		}
		score++
	} else if len(matcher.RequestBody.AllOf) > 0 {
		for _, condition := range matcher.RequestBody.AllOf {
			if !matchBodyCondition(exch.Request.Body, condition, systemNamespaces) {
				return NegativeMatchScore, false
			}
		}
		score += len(matcher.RequestBody.AllOf)
	} else if len(matcher.RequestBody.AnyOf) > 0 {
		matched := false
		for _, condition := range matcher.RequestBody.AnyOf {
			if matchBodyCondition(exch.Request.Body, condition, systemNamespaces) {
				matched = true
				break
			}
		}
		if !matched {
			return NegativeMatchScore, false
		}
		score++
	}

	// All expressions must match
	if len(matcher.AllOf) > 0 {
		for _, expr := range matcher.AllOf {
			// Evaluate the expression using the template engine
			result, err := evaluateExpression(expr.Expression, exch, imposterConfig, matcher)
			if err != nil {
				return NegativeMatchScore, false
			}
			if !expr.MatchCondition.Match(result) {
				return NegativeMatchScore, false
			}
		}
		score += len(matcher.AllOf)

		// At least one expression must match
	} else if len(matcher.AnyOf) > 0 {
		matched := false
		for _, expr := range matcher.AnyOf {
			// Evaluate the expression using the template engine
			result, err := evaluateExpression(expr.Expression, exch, imposterConfig, matcher)
			if err != nil {
				continue
			}
			if expr.MatchCondition.Match(result) {
				matched = true
				break
			}
		}
		if !matched {
			return NegativeMatchScore, false
		}
		score++
	}

	logger.Tracef("request %s %s base match score %d for matcher %v", r.Method, r.URL.Path, score, matcher)
	return score, isWildcard
}

// hasSingleBodyMatcher checks if a request matcher has a single body matcher
func hasSingleBodyMatcher(matcher *config.RequestMatcher) bool {
	return matcher.RequestBody.BodyMatchCondition != nil &&
		(matcher.RequestBody.Value != "" || matcher.RequestBody.JSONPath != "" || matcher.RequestBody.XPath != "")
}

// evaluateExpression evaluates a template expression in the context of the request
func evaluateExpression(expression string, exch *exchange.Exchange, imposterConfig *config.ImposterConfig, reqMatcher *config.RequestMatcher) (string, error) {
	// Simply evaluate the expression and return its value
	return template.ProcessTemplate(expression, exch, imposterConfig, reqMatcher), nil
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
			} else if best.Wildcard == m.Wildcard {
				// If both wildcard states are equal, prefer the one that is not runtime-generated
				if best.RuntimeGenerated && !m.RuntimeGenerated {
					best = m
					tie = false
				} else if best.RuntimeGenerated == m.RuntimeGenerated {
					tie = true
				}
			}
		} else if m.Score > best.Score {
			best = m
			tie = false
		}
	}

	return best, tie
}

// matchMixedSegment matches a segment that contains both path parameters and literal text
// For example: "{param}.diff" should match "123.diff" and extract param=123
// Returns: (matched bool, score int)
func matchMixedSegment(resourceSegment, requestSegment string, pathParams map[string]config.MatcherUnmarshaler) (bool, int) {
	// Parse the resource segment to extract parameters and build a proper regex
	pattern := ""
	paramNames := []string{}
	lastEnd := 0

	// Find all parameters in the segment and build regex incrementally
	for {
		start := strings.Index(resourceSegment[lastEnd:], "{")
		if start == -1 {
			// No more parameters, add remaining literal text
			if lastEnd < len(resourceSegment) {
				pattern += regexp.QuoteMeta(resourceSegment[lastEnd:])
			}
			break
		}
		start += lastEnd

		// Add literal text before the parameter
		if start > lastEnd {
			pattern += regexp.QuoteMeta(resourceSegment[lastEnd:start])
		}

		end := strings.Index(resourceSegment[start:], "}")
		if end == -1 {
			break
		}
		end += start

		paramName := resourceSegment[start+1 : end]
		paramNames = append(paramNames, paramName)

		// Add parameter capture group - match any character except slash
		pattern += "([^/]*?)"

		lastEnd = end + 1
	}

	if len(paramNames) == 0 {
		// No parameters found, shouldn't happen but handle gracefully
		return false, 0
	}

	// Create regex pattern from the segment
	regex, err := regexp.Compile("^" + pattern + "$")
	if err != nil {
		return false, 0
	}

	// Match against the request segment
	matches := regex.FindStringSubmatch(requestSegment)
	if matches == nil {
		return false, 0
	}

	// Validate extracted parameter values against path parameter conditions
	score := 0
	for i, paramName := range paramNames {
		paramValue := matches[i+1] // Skip the full match at index 0
		if condition, hasParam := pathParams[paramName]; hasParam {
			if !condition.Matcher.Match(paramValue) {
				return false, 0
			}
			score++
		}
	}

	// Base score for the mixed segment match plus any parameter condition matches
	// Mixed segments get higher score than pure parameters because they're more specific
	return true, 2 + score
}

// GetRequestBody reads and resets the request body
func GetRequestBody(r *http.Request) ([]byte, error) {
	// Handle nil body
	if r.Body == nil {
		return []byte{}, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// MatchXPath matches XML body content using XPath query
func MatchXPath(body []byte, condition config.BodyMatchCondition, systemNamespaces map[string]string) bool {
	// Merge system namespaces with condition namespaces, giving precedence to condition namespaces
	namespaces := make(map[string]string)
	for k, v := range systemNamespaces {
		namespaces[k] = v
	}
	for k, v := range condition.XMLNamespaces {
		namespaces[k] = v
	}

	result, success := query.XPathQuery(body, condition.XPath, namespaces)
	if !success {
		return false
	}
	return condition.Match(result)
}

// MatchJSONPath matches JSON body content using JSONPath query
func MatchJSONPath(body []byte, condition config.BodyMatchCondition) bool {
	result, success := query.JsonPathQuery(body, condition.JSONPath)
	if !success {
		return false
	}

	// Handle different result types
	switch v := result.(type) {
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
