package exchange

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExchange(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/orders", nil)
	body := []byte(`{"id":1}`)
	reqStore := store.NewRequestStore()
	state := &ResponseState{StatusCode: http.StatusOK}

	exch := NewExchange(req, body, reqStore, state)

	require.NotNil(t, exch.Request)
	assert.Same(t, req, exch.Request.Request)
	assert.Equal(t, body, exch.Request.Body)
	assert.Same(t, reqStore, exch.RequestStore)
	assert.Same(t, state, exch.ResponseState)
	// NewExchange does not populate the response context.
	assert.Nil(t, exch.Response)
}

func TestNewExchangeFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	reqStore := store.NewRequestStore()

	exch := NewExchangeFromRequest(req, nil, reqStore)

	assert.Same(t, req, exch.Request.Request)
	assert.Same(t, reqStore, exch.RequestStore)
	// The response state is left unset for callers to populate later.
	assert.Nil(t, exch.ResponseState)
}

func TestResponseState_SetStatusCode(t *testing.T) {
	rs := &ResponseState{}
	rs.SetStatusCode(http.StatusCreated)

	assert.Equal(t, http.StatusCreated, rs.StatusCode)
	assert.True(t, rs.StatusCodeSet, "explicit status codes must be flagged so configuration does not overwrite them")
}

func TestResponseState_SetBody(t *testing.T) {
	rs := &ResponseState{}
	rs.SetBody([]byte("hello"))

	assert.Equal(t, []byte("hello"), rs.Body)
	assert.True(t, rs.BodySet, "explicit bodies must be flagged so configuration does not overwrite them")
}

func TestResponseState_SetHeader(t *testing.T) {
	rs := &ResponseState{}
	// Header names are compared canonically, so casing differences refer to the
	// same header.
	rs.SetHeader("content-type", "application/json")

	assert.Equal(t, "application/json", rs.Headers["content-type"])
	assert.True(t, rs.IsHeaderExplicit("Content-Type"))
	assert.True(t, rs.IsHeaderExplicit("CONTENT-TYPE"))
	assert.False(t, rs.IsHeaderExplicit("X-Not-Set"))
}

func TestResponseState_ClearExplicitFlags(t *testing.T) {
	rs := &ResponseState{}
	rs.SetStatusCode(http.StatusTeapot)
	rs.SetBody([]byte("body"))
	rs.SetHeader("X-Custom", "value")

	rs.ClearExplicitFlags()

	assert.False(t, rs.StatusCodeSet)
	assert.False(t, rs.BodySet)
	assert.False(t, rs.IsHeaderExplicit("X-Custom"))
	// The values themselves are retained; only the explicit flags are cleared.
	assert.Equal(t, http.StatusTeapot, rs.StatusCode)
	assert.Equal(t, []byte("body"), rs.Body)
}

func TestResponseState_HandledWithResource(t *testing.T) {
	rs := &ResponseState{}
	resource := &config.BaseResource{}
	rs.HandledWithResource(resource)

	assert.True(t, rs.Handled)
	assert.Same(t, resource, rs.Resource)
}

func TestResponseState_WriteToResponseWriter(t *testing.T) {
	t.Run("writes status, headers and body", func(t *testing.T) {
		rs := &ResponseState{
			StatusCode: http.StatusAccepted,
			Headers:    map[string]string{"X-Custom": "value"},
			Body:       []byte("response body"),
		}
		rec := httptest.NewRecorder()

		rs.WriteToResponseWriter(rec)

		assert.Equal(t, http.StatusAccepted, rec.Code)
		assert.Equal(t, "value", rec.Header().Get("X-Custom"))
		assert.Equal(t, "response body", rec.Body.String())
	})

	t.Run("runs cleanup functions", func(t *testing.T) {
		cleaned := false
		rs := &ResponseState{
			StatusCode:       http.StatusOK,
			CleanupFunctions: []func(){func() { cleaned = true }, nil},
		}
		rec := httptest.NewRecorder()

		rs.WriteToResponseWriter(rec)

		assert.True(t, cleaned, "cleanup functions should run after writing; nil entries are skipped")
	})

	t.Run("hijacked responses skip writing but still clean up", func(t *testing.T) {
		cleaned := false
		rs := &ResponseState{
			Hijacked:         true,
			StatusCode:       http.StatusOK,
			Body:             []byte("should not be written"),
			CleanupFunctions: []func(){func() { cleaned = true }},
		}
		rec := httptest.NewRecorder()

		rs.WriteToResponseWriter(rec)

		assert.True(t, cleaned)
		// Nothing should have been written to the recorder.
		assert.Empty(t, rec.Body.String())
	})

	t.Run("declares and writes trailers", func(t *testing.T) {
		rs := &ResponseState{
			StatusCode: http.StatusOK,
			Body:       []byte("body"),
			Trailers:   map[string]string{"X-Checksum": "abc123"},
		}
		rec := httptest.NewRecorder()

		rs.WriteToResponseWriter(rec)

		// The trailer must be advertised via the Trailer header.
		assert.Equal(t, "X-Checksum", rec.Header().Get("Trailer"))
		assert.Equal(t, "abc123", rec.Header().Get("X-Checksum"))
	})
}
