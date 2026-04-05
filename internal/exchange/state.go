package exchange

import (
	"net/http"

	"github.com/imposter-project/imposter-go/internal/config"
)

// ResponseState tracks the state of the HTTP response
type ResponseState struct {
	StatusCode       int
	Headers          map[string]string
	Trailers         map[string]string    // HTTP/2 trailers, written after the body
	Body             []byte
	Stopped          bool                 // indicates if the response has been stopped (e.g., connection closed)
	Handled          bool                 // indicates if a handler has handled the request
	Resource         *config.BaseResource // the resource that handled the request
	Delay            config.Delay         // delay configuration for the response
	Fail             string               // failure type for the response
	File             string               // path to the response file
	CleanupFunctions []func()             // functions to execute after response is written
}

// HandledWithResource marks the response as handled and sets the resource that handled it
func (rs *ResponseState) HandledWithResource(resource *config.BaseResource) {
	rs.Handled = true
	rs.Resource = resource
}

// WriteToResponseWriter writes the final state to the http.ResponseWriter
func (rs *ResponseState) WriteToResponseWriter(w http.ResponseWriter) {
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
