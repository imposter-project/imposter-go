package matcher

import (
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
		requestStore       store.Store
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
			requestStore:       store.Store{},
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
			requestStore:       store.Store{},
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
			requestStore:       store.Store{},
			imposterConfig:     &config.ImposterConfig{},
			expectedScore:      NegativeMatchScore,
			expectedIsWildcard: false,
		},
		{
			name:               "empty conditions list",
			allOf:              []config.ExpressionMatchCondition{},
			request:            createTestRequest("GET", "/test", nil, nil),
			requestStore:       store.Store{},
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
			request:            createTestRequest("GET", "/test", nil, nil),
			requestStore:       store.Store{"user_role": "admin", "region": "EU"},
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

			score, isWildcard := CalculateMatchScore(matcher, tt.request, nil, map[string]string{}, tt.imposterConfig, &tt.requestStore)
			require.Equal(t, tt.expectedScore, score)
			require.Equal(t, tt.expectedIsWildcard, isWildcard)
		})
	}
}
