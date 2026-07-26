package emit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
)

// captureSink records every emitted body. When stopAfter is positive it stops
// accepting (returns false) once that many bodies have been recorded.
type captureSink struct {
	bodies    []string
	stopAfter int
}

func (s *captureSink) Emit(rs *exchange.ResponseState) bool {
	if s.stopAfter > 0 && len(s.bodies) >= s.stopAfter {
		return false
	}
	s.bodies = append(s.bodies, string(rs.Body))
	return true
}

func newExchange(t *testing.T, method, path string) (*exchange.Exchange, *store.Store) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rs := response.NewResponseState()
	st := store.NewRequestStore()
	return exchange.NewExchange(req, nil, st, rs), st
}

func contentProcessor() response.Processor {
	return response.NewProcessor(&config.ImposterConfig{}, "")
}

func TestEmitResponses_EmitsEachBodyInOrder(t *testing.T) {
	exch, _ := newExchange(t, http.MethodPost, "/chat")
	resps := []config.Response{
		{Content: "one"},
		{Content: "two"},
		{Content: "three"},
	}
	sink := &captureSink{}

	emitted := EmitResponses(exch, &config.RequestMatcher{}, resps, contentProcessor(), sink)

	if emitted != 3 {
		t.Fatalf("expected 3 emitted, got %d", emitted)
	}
	want := []string{"one", "two", "three"}
	for i, w := range want {
		if sink.bodies[i] != w {
			t.Errorf("body[%d] = %q, want %q", i, sink.bodies[i], w)
		}
	}
	if len(exch.ResponseState.Body) != 0 {
		t.Errorf("expected response body cleared after emit, got %q", exch.ResponseState.Body)
	}
}

func TestEmitResponses_StopsWhenSinkRejects(t *testing.T) {
	exch, _ := newExchange(t, http.MethodPost, "/chat")
	resps := []config.Response{{Content: "a"}, {Content: "b"}, {Content: "c"}}
	sink := &captureSink{stopAfter: 1}

	emitted := EmitResponses(exch, &config.RequestMatcher{}, resps, contentProcessor(), sink)

	if emitted != 1 {
		t.Fatalf("expected 1 emitted before sink rejected, got %d", emitted)
	}
	if len(sink.bodies) != 1 || sink.bodies[0] != "a" {
		t.Errorf("expected only first body emitted, got %v", sink.bodies)
	}
}

func TestEmitOne_EmptyBodyIsNoOp(t *testing.T) {
	rs := response.NewResponseState()
	sink := &captureSink{}
	if !EmitOne(rs, sink) {
		t.Error("EmitOne with empty body should report accepted")
	}
	if len(sink.bodies) != 0 {
		t.Errorf("expected no bodies emitted for empty response, got %v", sink.bodies)
	}
}

func TestDiscardSink_DropsBodies(t *testing.T) {
	rs := response.NewResponseState()
	rs.Body = []byte("dropped")
	if !EmitOne(rs, DiscardSink{}) {
		t.Error("DiscardSink should always accept")
	}
	if len(rs.Body) != 0 {
		t.Errorf("expected body cleared after EmitOne, got %q", rs.Body)
	}
}

func TestStreamHTTP_WritesChunksAndFlushes(t *testing.T) {
	exch, _ := newExchange(t, http.MethodPost, "/chat")
	rec := httptest.NewRecorder()
	exch.ResponseWriter = rec

	resource := &config.BaseResource{
		Responses: []config.Response{
			{Content: "chunk1", Headers: map[string]string{"Content-Type": "text/event-stream"}},
			{Content: "chunk2"},
		},
	}

	StreamHTTP(exch, resource, contentProcessor(), &config.ImposterConfig{}, "")

	if !exch.ResponseState.Streamed {
		t.Error("expected response state marked Streamed")
	}
	if got := rec.Body.String(); got != "chunk1chunk2" {
		t.Errorf("streamed body = %q, want %q", got, "chunk1chunk2")
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	if !rec.Flushed {
		t.Error("expected the writer to have been flushed")
	}
}

// nonFlushingWriter is an http.ResponseWriter that does not implement
// http.Flusher, exercising the buffered fallback path.
type nonFlushingWriter struct {
	header http.Header
	status int
	body   []byte
}

func (w *nonFlushingWriter) Header() http.Header    { return w.header }
func (w *nonFlushingWriter) WriteHeader(status int) { w.status = status }
func (w *nonFlushingWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}

func TestStreamHTTP_BufferedFallbackWhenNotFlushable(t *testing.T) {
	exch, _ := newExchange(t, http.MethodPost, "/chat")
	exch.ResponseWriter = &nonFlushingWriter{header: http.Header{}}

	resource := &config.BaseResource{
		Responses: []config.Response{
			{Content: "chunk1"},
			{Content: "chunk2"},
		},
	}

	StreamHTTP(exch, resource, contentProcessor(), &config.ImposterConfig{}, "")

	// The fallback concatenates into a single buffered body written the normal
	// way, so it must NOT mark the response Streamed.
	if exch.ResponseState.Streamed {
		t.Error("buffered fallback should not mark the response Streamed")
	}
	if got := string(exch.ResponseState.Body); got != "chunk1chunk2" {
		t.Errorf("buffered body = %q, want %q", got, "chunk1chunk2")
	}
}

func TestScheduleHost_FiresUntilLimit(t *testing.T) {
	sink := &captureSink{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	host := &ScheduleHost{
		Ctx:            ctx,
		ImposterConfig: &config.ImposterConfig{},
		RespProc:       contentProcessor(),
		Sink:           sink,
		Label:          "test",
		Schedules: []config.Schedule{
			{
				Every:    "10ms",
				Limit:    3,
				Response: &config.Response{Content: "tick"},
			},
		},
		NewExchange: func() *exchange.Exchange {
			req := httptest.NewRequest(http.MethodGet, "/events", nil)
			return exchange.NewExchange(req, nil, store.NewRequestStore(), response.NewResponseState())
		},
	}

	var wg sync.WaitGroup
	host.Start(&wg)
	wg.Wait()

	if len(sink.bodies) != 3 {
		t.Fatalf("expected 3 scheduled emissions (limit), got %d: %v", len(sink.bodies), sink.bodies)
	}
	for i, b := range sink.bodies {
		if b != "tick" {
			t.Errorf("emission[%d] = %q, want %q", i, b, "tick")
		}
	}
}

func TestScheduleHost_StopsOnContextCancel(t *testing.T) {
	sink := &captureSink{}
	ctx, cancel := context.WithCancel(context.Background())

	host := &ScheduleHost{
		Ctx:            ctx,
		ImposterConfig: &config.ImposterConfig{},
		RespProc:       contentProcessor(),
		Sink:           sink,
		Label:          "test",
		Schedules: []config.Schedule{
			{Every: "10ms", Response: &config.Response{Content: "tick"}}, // unlimited
		},
		NewExchange: func() *exchange.Exchange {
			req := httptest.NewRequest(http.MethodGet, "/events", nil)
			return exchange.NewExchange(req, nil, store.NewRequestStore(), response.NewResponseState())
		},
	}

	var wg sync.WaitGroup
	host.Start(&wg)
	time.Sleep(35 * time.Millisecond)
	cancel()
	wg.Wait() // must return promptly once the context is cancelled

	if len(sink.bodies) == 0 {
		t.Error("expected at least one emission before cancel")
	}
}
