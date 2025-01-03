package response

import (
	"fmt"
	"math/rand"
	"mime"
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
	Stopped    bool // indicates if the response has been stopped (e.g., connection closed)
	Handled    bool // indicates if a handler has handled the request
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
		// Mark the response as stopped to prevent writing the body
		rs.Stopped = true
		fmt.Printf("Handled request (simulated failure: CloseConnection) - method:%s, path:%s\n", r.Method, r.URL.Path)
		return true
	}
	return false
}

// ProcessResponse handles common response processing logic
func ProcessResponse(rs *ResponseState, req *http.Request, resp config.Response, configDir string, requestStore store.Store, imposterConfig *config.ImposterConfig) {
	// Handle delay if specified
	SimulateDelay(resp.Delay, req)

	// Set status code
	if resp.StatusCode > 0 {
		rs.StatusCode = resp.StatusCode
	}

	// Set resp headers
	for key, value := range resp.Headers {
		rs.Headers[key] = value
	}

	// Handle failure simulation
	if resp.Fail != "" {
		if SimulateFailure(rs, resp.Fail, req) {
			return
		}
	}

	// Get resp content
	var responseContent string
	if resp.File != "" {
		filePath := filepath.Join(configDir, resp.File)
		data, err := os.ReadFile(filePath)
		if err != nil {
			rs.StatusCode = http.StatusInternalServerError
			rs.Body = []byte("Failed to read file")
			return
		}
		responseContent = string(data)
	} else {
		responseContent = resp.Content
	}

	// Process template if enabled
	if resp.Template {
		responseContent = template.ProcessTemplate(responseContent, req, imposterConfig, requestStore)
	}

	// Set Content-Type header if not already set
	if _, exists := rs.Headers["Content-Type"]; !exists {
		// If response is from file, try to determine content type from extension
		if resp.File != "" {
			ext := filepath.Ext(resp.File)
			contentType := mime.TypeByExtension(ext)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			rs.Headers["Content-Type"] = contentType
			fmt.Printf("Inferred Content-Type %s from file extension %s\n", contentType, ext)
		} else {
			// If no file specified, assume JSON
			fmt.Println("No file extension available - assuming JSON content type")
			rs.Headers["Content-Type"] = "application/json"
		}
	}

	rs.Body = []byte(responseContent)
	fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
		req.Method, req.URL.Path, rs.StatusCode, len(responseContent))
}
