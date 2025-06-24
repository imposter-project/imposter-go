package response

import (
	"github.com/imposter-project/imposter-go/internal/exchange"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/imposter-project/imposter-go/pkg/utils"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/template"
)

const defaultIndexFile = "index.html"

// NewResponseState creates a new ResponseState with default values
func NewResponseState() *exchange.ResponseState {
	return &exchange.ResponseState{
		StatusCode: http.StatusOK,
		Headers:    make(map[string]string),
	}
}

// SimulateDelay simulates response delay based on the configuration
func SimulateDelay(delay config.Delay, r *http.Request) {
	if delay.Exact > 0 {
		logger.Infof("delaying request (exact: %dms) - method:%s, path:%s", delay.Exact, r.Method, r.URL.Path)
		time.Sleep(time.Duration(delay.Exact) * time.Millisecond)
	} else if delay.Min > 0 && delay.Max > 0 {
		actualDelay := rand.Intn(delay.Max-delay.Min+1) + delay.Min
		logger.Infof("delaying request (range: %dms-%dms, actual: %dms) - method:%s, path:%s",
			delay.Min, delay.Max, actualDelay, r.Method, r.URL.Path)
		time.Sleep(time.Duration(actualDelay) * time.Millisecond)
	}
}

// SimulateFailure simulates response failures based on the configuration
func SimulateFailure(rs *exchange.ResponseState, failureType string, r *http.Request) bool {
	switch failureType {
	case "EmptyResponse":
		rs.Body = nil
		logger.Infof("handled request (simulated failure: EmptyResponse) - method:%s, path:%s, status:%d, length:0",
			r.Method, r.URL.Path, rs.StatusCode)
		return true

	case "CloseConnection":
		rs.Stopped = true
		logger.Infof("handled request (simulated failure: CloseConnection) - method:%s, path:%s",
			r.Method, r.URL.Path)
		return true
	}
	return false
}

// processResponse handles common response processing logic
func processResponse(
	exch *exchange.Exchange,
	reqMatcher *config.RequestMatcher,
	resp *config.Response,
	configDir string,
	imposterConfig *config.ImposterConfig,
) {
	req := exch.Request.Request
	rs := exch.ResponseState

	// Handle delay if specified in ResponseState or Response config
	if rs.Delay.Exact > 0 || (rs.Delay.Min > 0 && rs.Delay.Max > 0) {
		SimulateDelay(rs.Delay, req)
	} else if resp.Delay.Exact > 0 || (resp.Delay.Min > 0 && resp.Delay.Max > 0) {
		SimulateDelay(resp.Delay, req)
	}

	// Set status code
	if resp.StatusCode > 0 {
		rs.StatusCode = resp.StatusCode
	}

	CopyResponseHeaders(resp.Headers, rs)

	// Handle failure simulation from ResponseState or Response config
	if rs.Fail != "" {
		if SimulateFailure(rs, rs.Fail, req) {
			return
		}
	} else if resp.Fail != "" {
		if SimulateFailure(rs, resp.Fail, req) {
			return
		}
	}

	// Handle response file or content
	respFile := resp.File
	respContent := resp.Content

	// Use file from ResponseState if set
	if rs.File != "" {
		respFile = rs.File
	}

	// Handle directory-based responses with wildcards
	if resp.Dir != "" {
		if reqMatcher == nil || !strings.HasSuffix(reqMatcher.Path, "/*") {
			logger.Errorf("directory response requires a wildcard path - method:%s, path:%s", req.Method, req.URL.Path)
			rs.StatusCode = http.StatusInternalServerError
			rs.Body = []byte("Invalid directory")
			return
		}
		basePath := strings.TrimSuffix(reqMatcher.Path, "*")
		requestSpecificPath := strings.TrimPrefix(req.URL.Path, basePath)

		if strings.HasSuffix(requestSpecificPath, "/") || requestSpecificPath == "" {
			requestSpecificPath += defaultIndexFile
		}

		// Set the response file path relative to the config directory
		respFile = filepath.Join(resp.Dir, requestSpecificPath)
		logger.Debugf("using directory-based response file: %s", respFile)
	}

	// Only override response content if specified, as it may have been set by an interceptor
	if respFile != "" || respContent != "" {
		var responseContent string
		if respFile != "" {
			filePath, err := utils.ValidatePath(respFile, configDir)
			if err != nil {
				rs.StatusCode = http.StatusInternalServerError
				rs.Body = []byte("Invalid file path")
				return
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				if os.IsNotExist(err) {
					logger.Errorf("response file not found: %s", filePath)
					rs.StatusCode = http.StatusNotFound
					return
				} else {
					logger.Errorf("error reading response file: %s", filePath)
					rs.StatusCode = http.StatusInternalServerError
					rs.Body = []byte("Failed to read response file")
					return
				}
			}
			responseContent = string(data)
		} else if respContent != "" {
			responseContent = respContent
		}

		// Process template if enabled
		if resp.Template {
			responseContent = template.ProcessTemplate(responseContent, exch, imposterConfig, reqMatcher)
		}

		rs.Body = []byte(responseContent)
	}

	if logger.IsTraceEnabled() {
		logger.Tracef("response headers: %v", rs.Headers)
		logger.Tracef("response body: %s", rs.Body)
	}

	SetContentTypeHeader(rs, respFile)

	logger.Debugf("updated response state - method:%s, path:%s, status:%d, length:%d",
		req.Method, req.URL.Path, rs.StatusCode, len(rs.Body))
}

// CopyResponseHeaders copies headers from a map to an exchange.ResponseState
// If a header already exists, it will be overwritten
func CopyResponseHeaders(src map[string]string, rs *exchange.ResponseState) {
	if src == nil {
		return
	}
	for key, value := range src {
		rs.Headers[key] = value
	}
}

// SetContentTypeHeader sets the Content-Type header based on the response file extension or defaults to JSON
func SetContentTypeHeader(rs *exchange.ResponseState, respFile string) {
	if _, exists := rs.Headers["Content-Type"]; !exists {
		// If response is from file, try to determine content type from extension
		if respFile != "" {
			ext := filepath.Ext(respFile)
			contentType := mime.TypeByExtension(ext)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			rs.Headers["Content-Type"] = contentType
			logger.Debugf("inferred Content-Type %s from file extension %s", contentType, ext)
		} else {
			// If no file specified, assume JSON
			logger.Infoln("no file extension available - assuming JSON content type")
			rs.Headers["Content-Type"] = "application/json"
		}
	}
}
