package matcher

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequestBody(t *testing.T) {
	t.Run("nil body returns empty slice", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.Body = nil

		body, err := GetRequestBody(req)
		require.NoError(t, err)
		assert.Empty(t, body)
	})

	t.Run("reads body and leaves it re-readable", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("payload")))

		body, err := GetRequestBody(req)
		require.NoError(t, err)
		assert.Equal(t, []byte("payload"), body)

		// The body must be reset so downstream handlers can read it again.
		again, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte("payload"), again)
	})
}

// TestMatchJSONPath_ResultTypes covers the non-string result branches of
// MatchJSONPath, where the resolved JSONPath value is coerced to a string
// before matching.
func TestMatchJSONPath_ResultTypes(t *testing.T) {
	tests := []struct {
		name      string
		body      []byte
		condition config.BodyMatchCondition
		want      bool
	}{
		{
			name: "numeric result matches stringified value",
			body: []byte(`{"age": 30}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{Value: "30"},
				JSONPath:       "$.age",
			},
			want: true,
		},
		{
			name: "boolean result matches stringified value",
			body: []byte(`{"active": true}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{Value: "true"},
				JSONPath:       "$.active",
			},
			want: true,
		},
		{
			name: "null result matches empty string",
			body: []byte(`{"deleted": null}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{Value: ""},
				JSONPath:       "$.deleted",
			},
			want: true,
		},
		{
			name: "array of non-strings does not match",
			body: []byte(`{"nums": [1, 2, 3]}`),
			condition: config.BodyMatchCondition{
				MatchCondition: config.MatchCondition{Value: "1"},
				JSONPath:       "$.nums",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MatchJSONPath(tt.body, tt.condition))
		})
	}
}

// TestMatchBodyCondition covers the dispatch in matchBodyCondition between
// JSONPath, XPath and plain literal body matching.
func TestMatchBodyCondition(t *testing.T) {
	t.Run("delegates to JSONPath", func(t *testing.T) {
		condition := config.BodyMatchCondition{
			MatchCondition: config.MatchCondition{Value: "Grace"},
			JSONPath:       "$.name",
		}
		got := matchBodyCondition([]byte(`{"name":"Grace"}`), condition, nil)
		assert.True(t, got)
	})

	t.Run("delegates to XPath", func(t *testing.T) {
		condition := config.BodyMatchCondition{
			MatchCondition: config.MatchCondition{Value: "Grace"},
			XPath:          "/user/name",
		}
		got := matchBodyCondition([]byte(`<user><name>Grace</name></user>`), condition, nil)
		assert.True(t, got)
	})

	t.Run("falls back to literal body match", func(t *testing.T) {
		condition := config.BodyMatchCondition{
			MatchCondition: config.MatchCondition{Value: "hello world"},
		}
		assert.True(t, matchBodyCondition([]byte("hello world"), condition, nil))
		assert.False(t, matchBodyCondition([]byte("goodbye"), condition, nil))
	})
}
