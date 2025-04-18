package matcher

import (
	"github.com/imposter-project/imposter-go/internal/exchange"
	"net/http"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/require"
)

func TestCalculateMatchScore_AllOfConditions(t *testing.T) {
	tests := []struct {
		name               string
		allOf              []config.ExpressionMatchCondition
		request            *http.Request
		requestStore       func() *store.Store
		imposterConfig     *config.ImposterConfig
		expectedScore      int
		expectedIsWildcard bool
	}{
		{
			name: "matches all conditions",
			allOf: []config.ExpressionMatchCondition{
				{
					Expression: "${context.request.headers.X-User-Role}",
					MatchCondition: config.MatchCondition{
						Value:    "admin",
						Operator: "EqualTo",
					},
				},
				{
					Expression: "${context.request.queryParams.region}",
					MatchCondition: config.MatchCondition{
						Value:    "EU",
						Operator: "EqualTo",
					},
				},
			},
			request: createTestRequest("GET", "/test?region=EU", nil, map[string]string{
				"X-User-Role": "admin",
			}),
			requestStore:       func() *store.Store { return store.NewRequestStore() },
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      2,
			expectedIsWildcard: false,
		},
		{
			name: "fails when one condition fails",
			allOf: []config.ExpressionMatchCondition{
				{
					Expression: "${context.request.headers.X-User-Role}",
					MatchCondition: config.MatchCondition{
						Value:    "admin",
						Operator: "EqualTo",
					},
				},
				{
					Expression: "${context.request.queryParams.region}",
					MatchCondition: config.MatchCondition{
						Value:    "EU",
						Operator: "EqualTo",
					},
				},
			},
			request: createTestRequest("GET", "/test?region=US", nil, map[string]string{
				"X-User-Role": "admin",
			}),
			requestStore:       func() *store.Store { return store.NewRequestStore() },
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      NegativeMatchScore,
			expectedIsWildcard: false,
		},
		{
			name: "fails when all conditions fail",
			allOf: []config.ExpressionMatchCondition{
				{
					Expression: "${context.request.headers.X-User-Role}",
					MatchCondition: config.MatchCondition{
						Value:    "admin",
						Operator: "EqualTo",
					},
				},
				{
					Expression: "${context.request.queryParams.region}",
					MatchCondition: config.MatchCondition{
						Value:    "EU",
						Operator: "EqualTo",
					},
				},
			},
			request: createTestRequest("GET", "/test?region=US", nil, map[string]string{
				"X-User-Role": "user",
			}),
			requestStore:       func() *store.Store { return store.NewRequestStore() },
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      NegativeMatchScore,
			expectedIsWildcard: false,
		},
		{
			name:               "empty conditions list",
			allOf:              []config.ExpressionMatchCondition{},
			request:            createTestRequest("GET", "/test", nil, nil),
			requestStore:       func() *store.Store { return store.NewRequestStore() },
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      0,
			expectedIsWildcard: false,
		},
		{
			name: "matches with store value",
			allOf: []config.ExpressionMatchCondition{
				{
					Expression: "${stores.request.user_role}",
					MatchCondition: config.MatchCondition{
						Value:    "admin",
						Operator: "EqualTo",
					},
				},
				{
					Expression: "${stores.request.region}",
					MatchCondition: config.MatchCondition{
						Value:    "EU",
						Operator: "EqualTo",
					},
				},
			},
			request: createTestRequest("GET", "/test", nil, nil),
			requestStore: func() *store.Store {
				s := store.NewRequestStore()
				s.StoreValue("user_role", "admin")
				s.StoreValue("region", "EU")
				return s
			},
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      2,
			expectedIsWildcard: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := &config.RequestMatcher{
				AllOf: tt.allOf,
			}

			exch := exchange.NewExchangeFromRequest(tt.request, nil, tt.requestStore())
			score, isWildcard := CalculateMatchScore(exch, matcher, map[string]string{}, tt.imposterConfig)
			require.Equal(t, tt.expectedScore, score)
			require.Equal(t, tt.expectedIsWildcard, isWildcard)
		})
	}
}
