package exchange

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"net/http"
)

// ResponseState tracks the state of the HTTP response
type ResponseState struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Stopped    bool                 // indicates if the response has been stopped (e.g., connection closed)
	Handled    bool                 // indicates if a handler has handled the request
	Resource   *config.BaseResource // the resource that handled the request
	Delay      config.Delay         // delay configuration for the response
	Fail       string               // failure type for the response
	File       string               // path to the response file
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

	for key, value := range rs.Headers {
		w.Header().Set(key, value)
	}
	w.WriteHeader(rs.StatusCode)
	if rs.Body != nil {
		w.Write(rs.Body)
	}
}
