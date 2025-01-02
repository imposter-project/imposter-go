package plugin

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/matcher"
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
		pathMatches, wildcard := matchPath(matcher.Path, r.URL.Path)
		if !pathMatches {
			return 0, false
		}
		score++
		isWildcard = wildcard
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
	}

	return score, isWildcard
}

// matchBodyCondition checks if a body condition matches the request body
func matchBodyCondition(body []byte, condition config.BodyMatchCondition) bool {
	if condition.JSONPath != "" {
		return matcher.MatchJSONPath(body, condition)
	} else if condition.XPath != "" {
		return matcher.MatchXPath(body, condition)
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

// matchPath checks if a request path matches a pattern, returns if it matches and if it's a wildcard match
func matchPath(pattern string, requestPath string) (matches bool, wildcard bool) {
	if strings.Contains(pattern, "*") {
		wildcard = true
		pattern = strings.ReplaceAll(pattern, "*", ".*")
		matched, _ := regexp.MatchString("^"+pattern+"$", requestPath)
		return matched, wildcard
	}
	return pattern == requestPath, wildcard
}
