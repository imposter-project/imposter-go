package passthrough

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
)

// newExchange builds an Exchange around an inbound request for testing.
func newExchange(req *http.Request, body []byte) *exchange.Exchange {
	return exchange.NewExchange(req, body, store.NewRequestStore(), response.NewResponseState())
}

func TestProxyForwardsRequestAndReturnsResponse(t *testing.T) {
	var capturedPath, capturedQuery, capturedBody string
	var capturedHeaders http.Header

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		capturedHeaders = r.Header.Clone()
		b, _ := io.ReadAll(r.Body)
		capturedBody = string(b)

		w.Header().Set("X-Upstream", "yes")
		w.Header().Set("Connection", "keep-alive") // hop-by-hop, must be stripped
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello from upstream"))
	}))
	defer upstream.Close()

	body := []byte(`{"k":"v"}`)
	req := httptest.NewRequest(http.MethodPost, "http://mock.local/api/users?page=2", strings.NewReader(string(body)))
	req.Header.Set("X-Trace-Id", "abc123")
	req.Header.Set("Accept-Encoding", "br") // hop-by-hop, must be stripped

	exch := newExchange(req, body)
	if err := Proxy(exch, "test", config.Upstream{URL: upstream.URL + "/base"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedPath != "/base/api/users" {
		t.Errorf("upstream path = %q, want %q", capturedPath, "/base/api/users")
	}
	if capturedQuery != "page=2" {
		t.Errorf("upstream query = %q, want %q", capturedQuery, "page=2")
	}
	if capturedBody != string(body) {
		t.Errorf("upstream body = %q, want %q", capturedBody, string(body))
	}
	if capturedHeaders.Get("X-Trace-Id") != "abc123" {
		t.Errorf("expected X-Trace-Id forwarded, headers: %v", capturedHeaders)
	}
	// The client's Accept-Encoding must not be forwarded. Go's transport may
	// transparently add its own "gzip" value, but our distinctive "br" must be gone.
	if strings.Contains(capturedHeaders.Get("Accept-Encoding"), "br") {
		t.Errorf("expected client Accept-Encoding stripped, got %q", capturedHeaders.Get("Accept-Encoding"))
	}

	rs := exch.ResponseState
	if rs.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", rs.StatusCode, http.StatusCreated)
	}
	if string(rs.Body) != "hello from upstream" {
		t.Errorf("body = %q, want %q", rs.Body, "hello from upstream")
	}
	if rs.Headers["X-Upstream"] != "yes" {
		t.Errorf("expected X-Upstream header forwarded, got %v", rs.Headers)
	}
	if _, ok := rs.Headers["Connection"]; ok {
		t.Errorf("expected Connection header stripped from response, got %v", rs.Headers)
	}
}

func TestProxyForwardsUpstreamErrorStatusVerbatim(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream boom"))
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "http://mock.local/thing", nil)
	exch := newExchange(req, nil)

	if err := Proxy(exch, "test", config.Upstream{URL: upstream.URL}); err != nil {
		t.Fatalf("an upstream 5xx must not be a transport error, got: %v", err)
	}
	if exch.ResponseState.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", exch.ResponseState.StatusCode, http.StatusInternalServerError)
	}
	if string(exch.ResponseState.Body) != "upstream boom" {
		t.Errorf("body = %q, want %q", exch.ResponseState.Body, "upstream boom")
	}
}

func TestProxyTransportFailureReturnsError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://mock.local/thing", nil)
	exch := newExchange(req, nil)

	// Port 1 is not listening; the dial should fail.
	if err := Proxy(exch, "test", config.Upstream{URL: "http://127.0.0.1:1"}); err == nil {
		t.Fatal("expected a transport error, got nil")
	}
}

func TestProxyForwardedHeaders(t *testing.T) {
	var captured http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "http://mock.local/thing", nil)
	req.RemoteAddr = "203.0.113.7:54321"
	req.Host = "mock.local"

	// Disabled by default.
	t.Setenv("IMPOSTER_PASSTHROUGH_FORWARDED_HEADERS", "")
	if err := Proxy(newExchange(req, nil), "test", config.Upstream{URL: upstream.URL}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.Get("X-Forwarded-For") != "" {
		t.Errorf("expected no X-Forwarded-For when disabled, got %q", captured.Get("X-Forwarded-For"))
	}

	// Enabled via env var.
	t.Setenv("IMPOSTER_PASSTHROUGH_FORWARDED_HEADERS", "true")
	if err := Proxy(newExchange(req, nil), "test", config.Upstream{URL: upstream.URL}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := captured.Get("X-Forwarded-For"); got != "203.0.113.7" {
		t.Errorf("X-Forwarded-For = %q, want %q", got, "203.0.113.7")
	}
	if got := captured.Get("X-Forwarded-Host"); got != "mock.local" {
		t.Errorf("X-Forwarded-Host = %q, want %q", got, "mock.local")
	}
	if got := captured.Get("X-Forwarded-Proto"); got != "http" {
		t.Errorf("X-Forwarded-Proto = %q, want %q", got, "http")
	}
	if got := captured.Get("Via"); got != "1.1 imposter" {
		t.Errorf("Via = %q, want %q", got, "1.1 imposter")
	}
}
