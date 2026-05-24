package passthrough

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

const defaultTimeout = 30 * time.Second

// HopByHopHeaders are headers that must not be forwarded in either direction.
// Per RFC 2616 §13.5.1, plus Accept-Encoding and Host, matching the JVM engine.
var HopByHopHeaders = map[string]struct{}{
	"Accept-Encoding":     {},
	"Host":                {},
	"Connection":          {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {},
	"Trailers":            {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

var (
	clientOnce sync.Once
	httpClient *http.Client
)

// getClient lazily constructs the shared HTTP client used for forwarding,
// honouring the IMPOSTER_PASSTHROUGH_TIMEOUT environment variable.
func getClient() *http.Client {
	clientOnce.Do(func() {
		timeout := defaultTimeout
		if raw := os.Getenv("IMPOSTER_PASSTHROUGH_TIMEOUT"); raw != "" {
			if d, err := time.ParseDuration(raw); err == nil {
				timeout = d
			} else {
				logger.Warnf("invalid IMPOSTER_PASSTHROUGH_TIMEOUT %q, using default %s", raw, defaultTimeout)
			}
		}
		httpClient = &http.Client{Timeout: timeout}
	})
	return httpClient
}

// forwardedHeadersEnabled reports whether X-Forwarded-* and Via headers should
// be injected into upstream requests.
func forwardedHeadersEnabled() bool {
	return strings.EqualFold(os.Getenv("IMPOSTER_PASSTHROUGH_FORWARDED_HEADERS"), "true")
}

// Proxy forwards the incoming request held in exch to the given upstream and
// writes the upstream response (status, headers, body) into exch.ResponseState.
// Normal response processing (templates, scripts, captures) is bypassed.
//
// A non-nil error indicates a transport-level failure reaching the upstream;
// the caller is responsible for translating that into a 502 response. A 4xx or
// 5xx status returned by the upstream is forwarded verbatim and is not an error.
func Proxy(exch *exchange.Exchange, upstream config.Upstream) error {
	srcReq := exch.Request.Request

	targetURL, err := JoinURL(upstream.URL, srcReq.URL.Path, srcReq.URL.RawQuery)
	if err != nil {
		return err
	}

	outReq, err := http.NewRequest(srcReq.Method, targetURL, bytes.NewReader(exch.Request.Body))
	if err != nil {
		return fmt.Errorf("failed to create upstream request: %w", err)
	}

	copyRequestHeaders(srcReq, outReq)
	if forwardedHeadersEnabled() {
		addForwardedHeaders(srcReq, outReq)
	}

	logger.Debugf("forwarding request to upstream %s", targetURL)
	resp, err := getClient().Do(outReq)
	if err != nil {
		return fmt.Errorf("failed to forward request to upstream %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read upstream response from %s: %w", targetURL, err)
	}

	rs := exch.ResponseState
	rs.StatusCode = resp.StatusCode
	copyResponseHeaders(resp.Header, rs)
	rs.Body = body
	return nil
}

// copyRequestHeaders copies all non-hop-by-hop headers from the incoming
// request onto the outgoing upstream request.
func copyRequestHeaders(src *http.Request, dst *http.Request) {
	for key, values := range src.Header {
		if isHopByHop(key) {
			continue
		}
		for _, v := range values {
			dst.Header.Add(key, v)
		}
	}
}

// copyResponseHeaders copies all non-hop-by-hop upstream response headers onto
// the response state. As ResponseState.Headers is single-valued, repeated
// header values are joined with ", ".
func copyResponseHeaders(src http.Header, rs *exchange.ResponseState) {
	if rs.Headers == nil {
		rs.Headers = make(map[string]string)
	}
	for key, values := range src {
		if isHopByHop(key) {
			continue
		}
		rs.Headers[key] = strings.Join(values, ", ")
	}
}

// addForwardedHeaders injects proxy-style headers so the upstream can identify
// the original client and protocol.
func addForwardedHeaders(src *http.Request, dst *http.Request) {
	if clientIP := clientIP(src.RemoteAddr); clientIP != "" {
		if prior := src.Header.Get("X-Forwarded-For"); prior != "" {
			dst.Header.Set("X-Forwarded-For", prior+", "+clientIP)
		} else {
			dst.Header.Set("X-Forwarded-For", clientIP)
		}
	}
	if src.Host != "" {
		dst.Header.Set("X-Forwarded-Host", src.Host)
	}
	proto := "http"
	if src.TLS != nil {
		proto = "https"
	}
	dst.Header.Set("X-Forwarded-Proto", proto)
	dst.Header.Set("Via", "1.1 imposter")
}

func clientIP(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}

func isHopByHop(header string) bool {
	_, ok := HopByHopHeaders[http.CanonicalHeaderKey(header)]
	return ok
}
