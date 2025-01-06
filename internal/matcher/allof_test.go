package matcher

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
)

func TestCalculateMatchScore_AllOf(t *testing.T) {
	tests := []struct {
		name             string
		matcher          config.RequestMatcher
		request          *http.Request
		body             []byte
		systemNamespaces map[string]string
		imposterConfig   *config.ImposterConfig
		requestStore     store.Store
		expectedScore    int
		expectedWildcard bool
	}{
		{
			name: "single expression matches",
			matcher: config.RequestMatcher{
				AllOf: []config.ExpressionMatchCondition{
					{
						Expression: "${stores.request.foo}",
						MatchCondition: config.MatchCondition{
							Value: "bar",
						},
					},
				},
			},
			request:        httptest.NewRequest(http.MethodGet, "/test", nil),
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.Store{"foo": "bar"},
			expectedScore:  1,
		},
		{
			name: "single expression does not match",
			matcher: config.RequestMatcher{
				AllOf: []config.ExpressionMatchCondition{
					{
						Expression: "${stores.request.foo}",
						MatchCondition: config.MatchCondition{
							Value: "bar",
						},
					},
				},
			},
			request:        httptest.NewRequest(http.MethodGet, "/test", nil),
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.Store{"foo": "not-bar"},
			expectedScore:  0,
		},
		{
			name: "multiple expressions all match",
			matcher: config.RequestMatcher{
				AllOf: []config.ExpressionMatchCondition{
					{
						Expression: "${stores.request.foo}",
						MatchCondition: config.MatchCondition{
							Value: "bar",
						},
					},
					{
						Expression: "${stores.request.baz}",
						MatchCondition: config.MatchCondition{
							Value:    "qux",
							Operator: "NotEqualTo",
						},
					},
				},
			},
			request:        httptest.NewRequest(http.MethodGet, "/test", nil),
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.Store{"foo": "bar", "baz": "not-qux"},
			expectedScore:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, wildcard := CalculateMatchScore(&tt.matcher, tt.request, tt.body, tt.systemNamespaces, tt.imposterConfig, tt.requestStore)
			if score != tt.expectedScore {
				t.Errorf("expected score %d, got %d", tt.expectedScore, score)
			}
			if wildcard != tt.expectedWildcard {
				t.Errorf("expected wildcard %v, got %v", tt.expectedWildcard, wildcard)
			}
		})
	}
}
