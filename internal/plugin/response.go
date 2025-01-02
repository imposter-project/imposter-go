package plugin

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
)

// ResponseState tracks the state of the HTTP response
type ResponseState struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Completed  bool // indicates if the response is complete (e.g., connection closed)
}

// NewResponseState creates a new ResponseState with default values
func NewResponseState() *ResponseState {
	return &ResponseState{
		StatusCode: http.StatusOK,
		Headers:    make(map[string]string),
	}
}

// WriteToResponseWriter writes the final state to the http.ResponseWriter
func (rs *ResponseState) WriteToResponseWriter(w http.ResponseWriter) {
	if rs.Completed {
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

// SimulateDelay simulates response delay based on the configuration
func SimulateDelay(delay config.Delay, r *http.Request) {
	if delay.Exact > 0 {
		fmt.Printf("Delaying request (exact: %dms) - method:%s, path:%s\n", delay.Exact, r.Method, r.URL.Path)
		time.Sleep(time.Duration(delay.Exact) * time.Millisecond)
	} else if delay.Min > 0 && delay.Max > 0 {
		actualDelay := rand.Intn(delay.Max-delay.Min+1) + delay.Min
		fmt.Printf("Delaying request (range: %dms-%dms, actual: %dms) - method:%s, path:%s\n",
			delay.Min, delay.Max, actualDelay, r.Method, r.URL.Path)
		time.Sleep(time.Duration(actualDelay) * time.Millisecond)
	}
}

// SimulateFailure simulates response failures based on the configuration
func SimulateFailure(rs *ResponseState, failureType string, r *http.Request) bool {
	switch failureType {
	case "EmptyResponse":
		// Send a status but no body
		rs.Body = nil
		fmt.Printf("Handled request (simulated failure: EmptyResponse) - method:%s, path:%s, status:%d, length:0\n",
			r.Method, r.URL.Path, rs.StatusCode)
		return true

	case "CloseConnection":
		// Mark the response as completed to prevent writing the body
		rs.Completed = true
		fmt.Printf("Handled request (simulated failure: CloseConnection) - method:%s, path:%s\n", r.Method, r.URL.Path)
		return true
	}
	return false
}
