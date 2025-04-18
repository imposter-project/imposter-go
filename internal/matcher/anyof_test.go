package matcher

import (
	"bytes"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/require"
)

func createTestRequest(method, path string, body []byte, headers map[string]string) *http.Request {
	u, _ := url.Parse("http://localhost" + path)
	r := &http.Request{
		Method: method,
		URL:    u,
		Header: make(http.Header),
	}
	if body != nil {
		r.Body = io.NopCloser(bytes.NewReader(body))
	} else {
		r.Body = http.NoBody
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func TestCalculateMatchScore_AnyOf(t *testing.T) {
	tests := []struct {
		name               string
		anyOf              []config.ExpressionMatchCondition
		request            *http.Request
		requestStore       func() *store.Store
		imposterConfig     *config.ImposterConfig
		expectedScore      int
		expectedIsWildcard bool
	}{
		{
			name: "matches first condition",
			anyOf: []config.ExpressionMatchCondition{
				{
					Expression: "${context.request.headers.Authorization}",
					MatchCondition: config.MatchCondition{
						Value:    "Bearer admin-token",
						Operator: "EqualTo",
					},
				},
				{
					Expression: "${context.request.queryParams.apiKey}",
					MatchCondition: config.MatchCondition{
						Value:    "secret-key",
						Operator: "EqualTo",
					},
				},
			},
			request: createTestRequest("GET", "/test", nil, map[string]string{
				"Authorization": "Bearer admin-token",
			}),
			requestStore:       func() *store.Store { return store.NewRequestStore() },
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      1,
			expectedIsWildcard: false,
		},
		{
			name: "matches second condition",
			anyOf: []config.ExpressionMatchCondition{
				{
					Expression: "${context.request.headers.Authorization}",
					MatchCondition: config.MatchCondition{
						Value:    "Bearer admin-token",
						Operator: "EqualTo",
					},
				},
				{
					Expression: "${context.request.queryParams.apiKey}",
					MatchCondition: config.MatchCondition{
						Value:    "secret-key",
						Operator: "EqualTo",
					},
				},
			},
			request:            createTestRequest("GET", "/test?apiKey=secret-key", nil, nil),
			requestStore:       func() *store.Store { return store.NewRequestStore() },
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      1,
			expectedIsWildcard: false,
		},
		{
			name: "matches both conditions",
			anyOf: []config.ExpressionMatchCondition{
				{
					Expression: "${context.request.headers.Authorization}",
					MatchCondition: config.MatchCondition{
						Value:    "Bearer admin-token",
						Operator: "EqualTo",
					},
				},
				{
					Expression: "${context.request.queryParams.apiKey}",
					MatchCondition: config.MatchCondition{
						Value:    "secret-key",
						Operator: "EqualTo",
					},
				},
			},
			request: createTestRequest("GET", "/test?apiKey=secret-key", nil, map[string]string{
				"Authorization": "Bearer admin-token",
			}),
			requestStore:       func() *store.Store { return store.NewRequestStore() },
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      1,
			expectedIsWildcard: false,
		},
		{
			name: "matches none of the conditions",
			anyOf: []config.ExpressionMatchCondition{
				{
					Expression: "${context.request.headers.Authorization}",
					MatchCondition: config.MatchCondition{
						Value:    "Bearer admin-token",
						Operator: "EqualTo",
					},
				},
				{
					Expression: "${context.request.queryParams.apiKey}",
					MatchCondition: config.MatchCondition{
						Value:    "secret-key",
						Operator: "EqualTo",
					},
				},
			},
			request: createTestRequest("GET", "/test?other=value", nil, map[string]string{
				"Authorization": "wrong-token",
			}),
			requestStore:       func() *store.Store { return store.NewRequestStore() },
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      NegativeMatchScore,
			expectedIsWildcard: false,
		},
		{
			name:               "empty conditions list",
			anyOf:              []config.ExpressionMatchCondition{},
			request:            createTestRequest("GET", "/test", nil, nil),
			requestStore:       func() *store.Store { return store.NewRequestStore() },
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      0,
			expectedIsWildcard: false,
		},
		{
			name: "matches with store value",
			anyOf: []config.ExpressionMatchCondition{
				{
					Expression: "${stores.request.user_role}",
					MatchCondition: config.MatchCondition{
						Value:    "admin",
						Operator: "EqualTo",
					},
				},
			},
			request: createTestRequest("GET", "/test", nil, nil),
			requestStore: func() *store.Store {
				s := store.NewRequestStore()
				s.StoreValue("user_role", "admin")
				return s
			},
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      1,
			expectedIsWildcard: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := &config.RequestMatcher{
				AnyOf: tt.anyOf,
			}

			exch := exchange.NewExchangeFromRequest(tt.request, nil, tt.requestStore())
			score, isWildcard := CalculateMatchScore(exch, matcher, map[string]string{}, tt.imposterConfig)
			require.Equal(t, tt.expectedScore, score)
			require.Equal(t, tt.expectedIsWildcard, isWildcard)
		})
	}
}
