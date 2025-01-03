package matcher

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMatchXPath(t *testing.T) {
	tests := []struct {
		name      string
		body      []byte
		condition config.BodyMatchCondition
		want      bool
	}{
		{
			name: "simple element match",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <user>Grace</user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//user",
			},
			want: true,
		},
		{
			name: "nested element match",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <users>
        <user>
            <name>Grace</name>
            <age>30</age>
        </user>
    </users>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//user/name",
			},
			want: true,
		},
		{
			name: "attribute match",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <user id="123">Grace</user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "123",
				},
				XPath: "//user/@id",
			},
			want: true,
		},
		{
			name: "with namespace",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root xmlns:ns="http://example.com">
    <ns:user>Grace</ns:user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//ns:user",
				XMLNamespaces: map[string]string{
					"ns": "http://example.com",
				},
			},
			want: true,
		},
		{
			name: "no match",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <user>Grace</user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//admin",
			},
			want: false,
		},
		{
			name: "invalid XML",
			body: []byte(`invalid xml`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//user",
			},
			want: false,
		},
		{
			name: "empty body",
			body: []byte(``),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				XPath: "//user",
			},
			want: false,
		},
		{
			name: "regex match",
			body: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<root>
    <user>Grace123</user>
</root>`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value:    "Grace\\d+",
					Operator: "Matches",
				},
				XPath: "//user",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchXPath(tt.body, tt.condition)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatchJSONPath(t *testing.T) {
	tests := []struct {
		name      string
		body      []byte
		condition config.BodyMatchCondition
		want      bool
	}{
		{
			name: "simple field match",
			body: []byte(`{"name": "Grace"}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				JSONPath: "$.name",
			},
			want: true,
		},
		{
			name: "nested field match",
			body: []byte(`{
				"user": {
					"name": "Grace",
					"age": 30
				}
			}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				JSONPath: "$.user.name",
			},
			want: true,
		},
		{
			name: "array element match",
			body: []byte(`{
				"users": [
					{"name": "Grace"},
					{"name": "Jane"}
				]
			}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				JSONPath: "$.users[0].name",
			},
			want: true,
		},
		{
			name: "array filter match",
			body: []byte(`{
				"users": [
					{"name": "Grace", "age": 30},
					{"name": "Jane", "age": 25}
				]
			}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				JSONPath: "$.users[?(@.age==30)].name",
			},
			want: true,
		},
		{
			name: "no match",
			body: []byte(`{"name": "Jane"}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				JSONPath: "$.name",
			},
			want: false,
		},
		{
			name: "invalid JSON",
			body: []byte(`invalid json`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				JSONPath: "$.name",
			},
			want: false,
		},
		{
			name: "empty body",
			body: []byte(``),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				JSONPath: "$.name",
			},
			want: false,
		},
		{
			name: "regex match",
			body: []byte(`{"id": "user123"}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value:    "user\\d+",
					Operator: "Matches",
				},
				JSONPath: "$.id",
			},
			want: true,
		},
		{
			name: "invalid JSONPath",
			body: []byte(`{"name": "Grace"}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{
					Value: "Grace",
				},
				JSONPath: "invalid[path",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchJSONPath(tt.body, tt.condition)
			assert.Equal(t, tt.want, got)
		})
	}
}

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
		name         string
		matcher      *config.RequestMatcher
		request      *http.Request
		body         []byte
		wantScore    int
		wantWildcard bool
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
			wantScore:    0,
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
			wantScore:    0,
			wantWildcard: false,
		},
		{
			name: "header match",
			matcher: &config.RequestMatcher{
				Headers: map[string]config.MatcherUnmarshaler{
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
			name: "simple body match",
			matcher: &config.RequestMatcher{
				RequestBody: config.RequestBody{
					BodyMatchCondition: config.BodyMatchCondition{
						MatchCondition: config.MatchCondition{Value: "test"},
					},
				},
			},
			request:      httptest.NewRequest("POST", "/test", strings.NewReader("test")),
			body:         []byte("test"),
			wantScore:    1,
			wantWildcard: false,
		},
		{
			name: "body JSONPath match",
			matcher: &config.RequestMatcher{
				RequestBody: config.RequestBody{
					BodyMatchCondition: config.BodyMatchCondition{
						MatchCondition: config.MatchCondition{Value: "Grace"},
						JSONPath:       "$.name",
					},
				},
			},
			request:      httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"Grace"}`)),
			body:         []byte(`{"name":"Grace"}`),
			wantScore:    1,
			wantWildcard: false,
		},
		{
			name: "body XPath match",
			matcher: &config.RequestMatcher{
				RequestBody: config.RequestBody{
					BodyMatchCondition: config.BodyMatchCondition{
						MatchCondition: config.MatchCondition{Value: "Grace"},
						XPath:          "//name",
					},
				},
			},
			request:      httptest.NewRequest("POST", "/test", strings.NewReader(`<user><name>Grace</name></user>`)),
			body:         []byte(`<user><name>Grace</name></user>`),
			wantScore:    1,
			wantWildcard: false,
		},
		{
			name: "body AllOf match",
			matcher: &config.RequestMatcher{
				RequestBody: config.RequestBody{
					AllOf: []config.BodyMatchCondition{
						{
							MatchCondition: config.MatchCondition{Value: "Grace"},
							JSONPath:       "$.name",
						},
						{
							MatchCondition: config.MatchCondition{Value: "30"},
							JSONPath:       "$.age",
						},
					},
				},
			},
			request:      httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"Grace","age":30}`)),
			body:         []byte(`{"name":"Grace","age":30}`),
			wantScore:    1,
			wantWildcard: false,
		},
		{
			name: "body AnyOf match",
			matcher: &config.RequestMatcher{
				RequestBody: config.RequestBody{
					AnyOf: []config.BodyMatchCondition{
						{
							MatchCondition: config.MatchCondition{Value: "Grace"},
							JSONPath:       "$.name",
						},
						{
							MatchCondition: config.MatchCondition{Value: "Jane"},
							JSONPath:       "$.name",
						},
					},
				},
			},
			request:      httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"Grace"}`)),
			body:         []byte(`{"name":"Grace"}`),
			wantScore:    1,
			wantWildcard: false,
		},
		{
			name: "multiple criteria match",
			matcher: &config.RequestMatcher{
				Method: "POST",
				Path:   "/users",
				Headers: map[string]config.MatcherUnmarshaler{
					"Content-Type": {Matcher: config.StringMatcher("application/json")},
				},
				RequestBody: config.RequestBody{
					BodyMatchCondition: config.BodyMatchCondition{
						MatchCondition: config.MatchCondition{Value: "Grace"},
						JSONPath:       "$.name",
					},
				},
			},
			request: func() *http.Request {
				r := httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"Grace"}`))
				r.Header.Set("Content-Type", "application/json")
				return r
			}(),
			body:         []byte(`{"name":"Grace"}`),
			wantScore:    4,
			wantWildcard: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotScore, gotWildcard := CalculateMatchScore(tt.matcher, tt.request, tt.body)
			assert.Equal(t, tt.wantScore, gotScore)
			assert.Equal(t, tt.wantWildcard, gotWildcard)
		})
	}
}
