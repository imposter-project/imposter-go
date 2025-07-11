package matcher

import (
	"github.com/imposter-project/imposter-go/internal/exchange"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestFindBestMatch(t *testing.T) {
	tests := []struct {
		name     string
		matches  []MatchResult
		wantBest MatchResult
		wantTie  bool
	}{
		{
			name:     "empty matches",
			matches:  []MatchResult{},
			wantBest: MatchResult{},
			wantTie:  false,
		},
		{
			name: "single match",
			matches: []MatchResult{
				{Resource: &config.Resource{}, Score: 2},
			},
			wantBest: MatchResult{Resource: &config.Resource{}, Score: 2},
			wantTie:  false,
		},
		{
			name: "higher score wins",
			matches: []MatchResult{
				{Resource: &config.Resource{}, Score: 2},
				{Resource: &config.Resource{}, Score: 3},
				{Resource: &config.Resource{}, Score: 1},
			},
			wantBest: MatchResult{Resource: &config.Resource{}, Score: 3},
			wantTie:  false,
		},
		{
			name: "non-wildcard preferred over wildcard with same score",
			matches: []MatchResult{
				{Resource: &config.Resource{}, Score: 2, Wildcard: true},
				{Resource: &config.Resource{}, Score: 2, Wildcard: false},
			},
			wantBest: MatchResult{Resource: &config.Resource{}, Score: 2, Wildcard: false},
			wantTie:  false,
		},
		{
			name: "tie with same score and both non-wildcard",
			matches: []MatchResult{
				{Resource: &config.Resource{}, Score: 2, Wildcard: false},
				{Resource: &config.Resource{}, Score: 2, Wildcard: false},
			},
			wantBest: MatchResult{Resource: &config.Resource{}, Score: 2, Wildcard: false},
			wantTie:  true,
		},
		{
			name: "tie with same score and both wildcard",
			matches: []MatchResult{
				{Resource: &config.Resource{}, Score: 2, Wildcard: true},
				{Resource: &config.Resource{}, Score: 2, Wildcard: true},
			},
			wantBest: MatchResult{Resource: &config.Resource{}, Score: 2, Wildcard: true},
			wantTie:  true,
		},
		{
			name: "interceptor match",
			matches: []MatchResult{
				{Interceptor: &config.Interceptor{}, Score: 2},
				{Interceptor: &config.Interceptor{}, Score: 3},
			},
			wantBest: MatchResult{Interceptor: &config.Interceptor{}, Score: 3},
			wantTie:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBest, gotTie := FindBestMatch(tt.matches)
			assert.Equal(t, tt.wantBest.Score, gotBest.Score)
			assert.Equal(t, tt.wantBest.Wildcard, gotBest.Wildcard)
			assert.Equal(t, tt.wantTie, gotTie)
		})
	}
}

func TestCalculateMatchScore(t *testing.T) {
	tests := []struct {
		name             string
		matcher          *config.RequestMatcher
		request          *http.Request
		body             []byte
		systemNamespaces map[string]string
		imposterConfig   *config.ImposterConfig
		requestStore     store.Store
		wantScore        int
		wantWildcard     bool
	}{
		{
			name: "method match only",
			matcher: &config.RequestMatcher{
				Method: "POST",
			},
			request:      httptest.NewRequest("POST", "/test", nil),
			wantScore:    1,
			wantWildcard: false,
		},
		{
			name: "method mismatch",
			matcher: &config.RequestMatcher{
				Method: "POST",
			},
			request:      httptest.NewRequest("GET", "/test", nil),
			wantScore:    NegativeMatchScore,
			wantWildcard: false,
		},
		{
			name: "exact path match",
			matcher: &config.RequestMatcher{
				Path: "/test/path",
			},
			request:      httptest.NewRequest("GET", "/test/path", nil),
			wantScore:    2,
			wantWildcard: false,
		},
		{
			name: "wildcard path match",
			matcher: &config.RequestMatcher{
				Path: "/test/*",
			},
			request:      httptest.NewRequest("GET", "/test/anything", nil),
			wantScore:    1,
			wantWildcard: true,
		},
		{
			name: "path parameter match",
			matcher: &config.RequestMatcher{
				Path: "/users/{id}",
				PathParams: map[string]config.MatcherUnmarshaler{
					"id": {Matcher: config.StringMatcher("123")},
				},
			},
			request:      httptest.NewRequest("GET", "/users/123", nil),
			wantScore:    2,
			wantWildcard: false,
		},
		{
			name: "path parameter mismatch",
			matcher: &config.RequestMatcher{
				Path: "/users/{id}",
				PathParams: map[string]config.MatcherUnmarshaler{
					"id": {Matcher: config.StringMatcher("123")},
				},
			},
			request:      httptest.NewRequest("GET", "/users/456", nil),
			wantScore:    NegativeMatchScore,
			wantWildcard: false,
		},
		{
			name: "header match",
			matcher: &config.RequestMatcher{
				RequestHeaders: map[string]config.MatcherUnmarshaler{
					"Content-Type": {Matcher: config.StringMatcher("application/json")},
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("GET", "/test", nil)
				r.Header.Set("Content-Type", "application/json")
				return r
			}(),
			wantScore:    1,
			wantWildcard: false,
		},
		{
			name: "query param match",
			matcher: &config.RequestMatcher{
				QueryParams: map[string]config.MatcherUnmarshaler{
					"filter": {Matcher: config.StringMatcher("active")},
				},
			},
			request:      httptest.NewRequest("GET", "/test?filter=active", nil),
			wantScore:    1,
			wantWildcard: false,
		},
		{
			name: "form param match",
			matcher: &config.RequestMatcher{
				FormParams: map[string]config.MatcherUnmarshaler{
					"username": {Matcher: config.StringMatcher("john")},
				},
			},
			request: func() *http.Request {
				form := url.Values{}
				form.Add("username", "john")
				r := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return r
			}(),
			wantScore:    1,
			wantWildcard: false,
		},
		{
			name: "multiple criteria match",
			matcher: &config.RequestMatcher{
				Method: "POST",
				Path:   "/users",
				RequestHeaders: map[string]config.MatcherUnmarshaler{
					"Content-Type": {Matcher: config.StringMatcher("application/json")},
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("POST", "/users", nil)
				r.Header.Set("Content-Type", "application/json")
				return r
			}(),
			wantScore:    3,
			wantWildcard: false,
		},
		{
			name: "mixed path parameter match - simple",
			matcher: &config.RequestMatcher{
				Path: "/example/{param}.diff",
			},
			request:      httptest.NewRequest("GET", "/example/123.diff", nil),
			wantScore:    3, // 1 for /example + 2 for mixed segment {param}.diff
			wantWildcard: false,
		},
		{
			name: "mixed path parameter with condition match",
			matcher: &config.RequestMatcher{
				Path: "/files/{name}.{ext}",
				PathParams: map[string]config.MatcherUnmarshaler{
					"ext": {Matcher: config.StringMatcher("pdf")},
				},
			},
			request:      httptest.NewRequest("GET", "/files/document.pdf", nil),
			wantScore:    4, // 1 for /files + 2 for mixed segment + 1 for parameter condition
			wantWildcard: false,
		},
		{
			name: "mixed path parameter with condition mismatch",
			matcher: &config.RequestMatcher{
				Path: "/files/{name}.{ext}",
				PathParams: map[string]config.MatcherUnmarshaler{
					"ext": {Matcher: config.StringMatcher("pdf")},
				},
			},
			request:      httptest.NewRequest("GET", "/files/document.txt", nil),
			wantScore:    NegativeMatchScore,
			wantWildcard: false,
		},
		{
			name: "mixed path parameter vs pure parameter - more specific wins",
			matcher: &config.RequestMatcher{
				Path: "/example/{param}.diff",
			},
			request:      httptest.NewRequest("GET", "/example/123.diff", nil),
			wantScore:    3, // Mixed segment should score higher than pure parameter
			wantWildcard: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exch := exchange.NewExchangeFromRequest(tt.request, tt.body, store.NewRequestStore())
			gotScore, gotWildcard := CalculateMatchScore(exch, tt.matcher, tt.systemNamespaces, tt.imposterConfig)
			if gotScore != tt.wantScore {
				t.Errorf("expected score %d, got %d", tt.wantScore, gotScore)
			}
			if gotWildcard != tt.wantWildcard {
				t.Errorf("expected wildcard %v, got %v", tt.wantWildcard, gotWildcard)
			}
		})
	}
}

func TestMixedParameterPriority(t *testing.T) {
	// Test the specific case where:
	// /example/{param}.diff should be preferred over /example/{param}
	// when the request is /example/123.diff

	request := httptest.NewRequest("GET", "/example/123.diff", nil)
	exch := exchange.NewExchangeFromRequest(request, nil, store.NewRequestStore())

	// Mixed parameter matcher - more specific
	mixedMatcher := &config.RequestMatcher{
		Path: "/example/{param}.diff",
	}
	mixedScore, _ := CalculateMatchScore(exch, mixedMatcher, nil, nil)

	// Pure parameter matcher - less specific
	pureMatcher := &config.RequestMatcher{
		Path: "/example/{param}",
	}
	pureScore, _ := CalculateMatchScore(exch, pureMatcher, nil, nil)

	// The mixed parameter matcher should score higher than the pure parameter matcher
	assert.Greater(t, mixedScore, pureScore, "Mixed parameter matcher should score higher than pure parameter matcher")
	assert.Greater(t, mixedScore, 0, "Mixed parameter matcher should have positive score")
	assert.Greater(t, pureScore, 0, "Pure parameter matcher should have positive score")

	t.Logf("Mixed parameter score: %d, Pure parameter score: %d", mixedScore, pureScore)
}
