package matcher

import (
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/stretchr/testify/assert"
)

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
