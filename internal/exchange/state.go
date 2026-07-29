package exchange

import (
	"net/http"

	"github.com/imposter-project/imposter-go/internal/config"
)

// ResponseState tracks the state of the HTTP response
type ResponseState struct {
	StatusCode       int
	Headers          map[string]string
	Trailers         map[string]string // HTTP/2 trailers, written after the body
	Body             []byte
	Stopped          bool                 // indicates if the response has been stopped (e.g., connection closed)
	Handled          bool                 // indicates if a handler has handled the request
	Hijacked         bool                 // indicates a plugin has taken over the underlying connection (e.g. websocket upgrade)
	Resource         *config.BaseResource // the resource that handled the request
	Delay            config.Delay         // delay configuration for the response
	Fail             string               // failure type for the response
	File             string               // path to the response file
	CleanupFunctions []func()             // functions to execute after response is written

	// StatusCodeSet indicates the status code was set explicitly by a script,
	// rather than defaulted. Config values are only applied as a fallback, so
	// a script-set status code is not overwritten by a resource's response
	// configuration.
	StatusCodeSet bool

	// BodySet indicates the body was set explicitly by a script. As with
	// StatusCodeSet, a resource's response content is only used as a fallback.
	BodySet bool

	// explicitHeaders holds the canonical names of headers set explicitly by a
	// script, which are not overwritten by a resource's response configuration.
	// Headers set by a plugin, such as the SOAP content type, are not tracked
	// here, so configuration continues to take precedence over those.
	explicitHeaders map[string]bool
}

// HandledWithResource marks the response as handled and sets the resource that handled it
func (rs *ResponseState) HandledWithResource(resource *config.BaseResource) {
	rs.Handled = true
	rs.Resource = resource
}

// SetStatusCode records an explicitly requested status code, marking it so
// that it takes precedence over any status code in the response configuration.
func (rs *ResponseState) SetStatusCode(statusCode int) {
	rs.StatusCode = statusCode
	rs.StatusCodeSet = true
}

// SetBody records an explicitly requested body, marking it so that it takes
// precedence over any content in the response configuration.
func (rs *ResponseState) SetBody(body []byte) {
	rs.Body = body
	rs.BodySet = true
}

// SetHeader records an explicitly requested header, marking it so that it
// takes precedence over a header of the same name in the response
// configuration. Names are compared in their canonical form, so a script and a
// configuration that differ only in casing still refer to the same header.
func (rs *ResponseState) SetHeader(name, value string) {
	if rs.Headers == nil {
		rs.Headers = make(map[string]string)
	}
	rs.Headers[name] = value

	if rs.explicitHeaders == nil {
		rs.explicitHeaders = make(map[string]bool)
	}
	rs.explicitHeaders[http.CanonicalHeaderKey(name)] = true
}

// IsHeaderExplicit reports whether the named header was set explicitly, and so
// should not be overwritten by the response configuration.
func (rs *ResponseState) IsHeaderExplicit(name string) bool {
	return rs.explicitHeaders[http.CanonicalHeaderKey(name)]
}

// ClearExplicitFlags forgets that the status code, body and headers were
// explicitly set, allowing a subsequent configured response to be applied
// unconditionally. This is used where the configured response must win
// regardless of what came before it, such as a rate limit response.
func (rs *ResponseState) ClearExplicitFlags() {
	rs.StatusCodeSet = false
	rs.BodySet = false
	rs.explicitHeaders = nil
}

// WriteToResponseWriter writes the final state to the http.ResponseWriter
func (rs *ResponseState) WriteToResponseWriter(w http.ResponseWriter) {
	if rs.Hijacked {
		// The connection has been taken over (e.g. websocket upgrade); nothing
		// may be written, but cleanup functions still run.
		for _, cleanup := range rs.CleanupFunctions {
			if cleanup != nil {
				cleanup()
			}
		}
		return
	}

	if rs.Stopped {
		// Handle connection closing
		if hijacker, ok := w.(http.Hijacker); ok {
			if conn, _, err := hijacker.Hijack(); err == nil {
				conn.Close()
				return
			}
		}
		// Fallback if hijacking is not supported
		rs.StatusCode = http.StatusInternalServerError
		rs.Body = []byte("HTTP server does not support connection hijacking")
	}

	// Declare any trailers before WriteHeader so they are advertised in the
	// Trailer response header. net/http will then recognise trailer values
	// set after the body is written.
	for key := range rs.Trailers {
		w.Header().Add("Trailer", key)
	}

	for key, value := range rs.Headers {
		w.Header().Set(key, value)
	}
	w.WriteHeader(rs.StatusCode)
	if rs.Body != nil {
		w.Write(rs.Body)
	}

	// Write trailer values after the body
	for key, value := range rs.Trailers {
		w.Header().Set(key, value)
	}

	// Execute cleanup functions after response is written
	for _, cleanup := range rs.CleanupFunctions {
		if cleanup != nil {
			cleanup()
		}
	}
}
