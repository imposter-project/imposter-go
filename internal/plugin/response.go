package plugin

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/internal/template"
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

// ProcessResponse handles common response processing logic
func ProcessResponse(rs *ResponseState, r *http.Request, response config.Response, configDir string, requestStore store.Store, imposterConfig *config.ImposterConfig) {
	// Handle delay if specified
	SimulateDelay(response.Delay, r)

	// Set status code
	if response.StatusCode > 0 {
		rs.StatusCode = response.StatusCode
	}

	// Set response headers
	for key, value := range response.Headers {
		rs.Headers[key] = value
	}

	// Handle failure simulation
	if response.Fail != "" {
		if SimulateFailure(rs, response.Fail, r) {
			return
		}
	}

	// Get response content
	var responseContent string
	if response.File != "" {
		filePath := filepath.Join(configDir, response.File)
		data, err := os.ReadFile(filePath)
		if err != nil {
			rs.StatusCode = http.StatusInternalServerError
			rs.Body = []byte("Failed to read file")
			return
		}
		responseContent = string(data)
	} else {
		responseContent = response.Content
	}

	// Process template if enabled
	if response.Template {
		responseContent = template.ProcessTemplate(responseContent, r, imposterConfig, requestStore)
	}

	rs.Body = []byte(responseContent)
	fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
		r.Method, r.URL.Path, rs.StatusCode, len(responseContent))
}
