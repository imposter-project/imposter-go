package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMatchCondition_Match(t *testing.T) {
	tests := []struct {
		name        string
		condition   MatchCondition
		actualValue string
		want        bool
	}{
		{
			name:        "EqualTo - explicit operator",
			condition:   MatchCondition{Value: "test", Operator: "EqualTo"},
			actualValue: "test",
			want:        true,
		},
		{
			name:        "EqualTo - implicit operator",
			condition:   MatchCondition{Value: "test"},
			actualValue: "test",
			want:        true,
		},
		{
			name:        "NotEqualTo",
			condition:   MatchCondition{Value: "test", Operator: "NotEqualTo"},
			actualValue: "other",
			want:        true,
		},
		{
			name:        "Exists",
			condition:   MatchCondition{Operator: "Exists"},
			actualValue: "any value",
			want:        true,
		},
		{
			name:        "NotExists",
			condition:   MatchCondition{Operator: "NotExists"},
			actualValue: "",
			want:        true,
		},
		{
			name:        "Contains",
			condition:   MatchCondition{Value: "world", Operator: "Contains"},
			actualValue: "hello world",
			want:        true,
		},
		{
			name:        "NotContains",
			condition:   MatchCondition{Value: "world", Operator: "NotContains"},
			actualValue: "hello",
			want:        true,
		},
		{
			name:        "Matches",
			condition:   MatchCondition{Value: "^test\\d+$", Operator: "Matches"},
			actualValue: "test123",
			want:        true,
		},
		{
			name:        "Match success with character class subtraction",
			condition:   MatchCondition{Value: "[A-Z-[BC]]", Operator: "Matches"},
			actualValue: "A",
			want:        true,
		},
		{
			name:        "Match failure with character class subtraction",
			condition:   MatchCondition{Value: "[A-Z-[BC]]", Operator: "Matches"},
			actualValue: "B",
			want:        false,
		},
		{
			name:        "NotMatches",
			condition:   MatchCondition{Value: "^test\\d+$", Operator: "NotMatches"},
			actualValue: "invalid",
			want:        true,
		},
		{
			name:        "Invalid operator",
			condition:   MatchCondition{Value: "test", Operator: "InvalidOp"},
			actualValue: "test",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.condition.Match(tt.actualValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStringMatcher_Match(t *testing.T) {
	tests := []struct {
		name        string
		matcher     StringMatcher
		actualValue string
		want        bool
	}{
		{
			name:        "exact match",
			matcher:     StringMatcher("test"),
			actualValue: "test",
			want:        true,
		},
		{
			name:        "no match",
			matcher:     StringMatcher("test"),
			actualValue: "other",
			want:        false,
		},
		{
			name:        "empty matcher",
			matcher:     StringMatcher(""),
			actualValue: "",
			want:        true,
		},
		{
			name:        "case sensitive",
			matcher:     StringMatcher("Test"),
			actualValue: "test",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.matcher.Match(tt.actualValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBodyMatchCondition_Match(t *testing.T) {
	tests := []struct {
		name        string
		condition   BodyMatchCondition
		actualValue string
		want        bool
	}{
		{
			name: "with JSONPath",
			condition: BodyMatchCondition{
				MatchCondition: MatchCondition{Value: "test", Operator: "EqualTo"},
				JSONPath:       "$.name",
			},
			actualValue: "test",
			want:        true,
		},
		{
			name: "with XPath",
			condition: BodyMatchCondition{
				MatchCondition: MatchCondition{Value: "test", Operator: "Contains"},
				XPath:          "//name",
				XMLNamespaces:  map[string]string{"ns": "http://example.com"},
			},
			actualValue: "test",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.condition.Match(tt.actualValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWebSocketConfigUnmarshal(t *testing.T) {
	// 'on' must parse as a string key/value with yaml.v3 (which dropped the
	// YAML 1.1 on/off boolean resolution), without needing quotes.
	yamlContent := `plugin: websocket
resources:
  - path: /gateway
    on: open
    response:
      content: challenge
    schedule:
      - every: 15s
        response:
          content: tick
  - path: /gateway
    requestBody:
      jsonPath: $.method
      value: connect
    responses:
      - content: first
      - content: second
        delay:
          exact: 250
schedules:
  - name: job
    cron: "0 * * * *"
    steps:
      - type: remote
        url: http://example.com
`

	var cfg Config
	err := yaml.Unmarshal([]byte(yamlContent), &cfg)
	require.NoError(t, err)

	require.Len(t, cfg.Resources, 2)
	require.Equal(t, "open", cfg.Resources[0].On)
	require.Equal(t, WebSocketEventOpen, cfg.Resources[0].NormalisedOn())
	require.Len(t, cfg.Resources[0].Schedule, 1)
	require.Equal(t, "15s", cfg.Resources[0].Schedule[0].Every)
	require.Equal(t, "tick", cfg.Resources[0].Schedule[0].Response.Content)

	require.Empty(t, cfg.Resources[1].On)
	require.Equal(t, WebSocketEventMessage, cfg.Resources[1].NormalisedOn())
	require.Len(t, cfg.Resources[1].Responses, 2)
	require.Equal(t, 250, cfg.Resources[1].Responses[1].Delay.Exact)

	require.Len(t, cfg.Schedules, 1)
	require.Equal(t, "0 * * * *", cfg.Schedules[0].Cron)
	require.Len(t, cfg.Schedules[0].Steps, 1)
}
