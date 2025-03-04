package handler

import (
	"net/http"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/pkg/logger"

	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/plugin"
)

// HandleRequest processes incoming HTTP requests and routes them to the appropriate handler
func HandleRequest(imposterConfig *config.ImposterConfig, w http.ResponseWriter, req *http.Request, plugins []plugin.Plugin) {
	// Check for CORS configuration in any of the plugins
	for _, plg := range plugins {
		if plg.GetConfig().Cors != nil {
			// If this was a preflight request that was handled, return early
			if handled := handleCORS(w, req, plg.GetConfig().Cors); handled {
				logger.Infof("handled CORS preflight request - method:%s, path:%s", req.Method, req.URL.Path)
				return
			}
			// We found a CORS config, no need to check others
			break
		}
	}

	// Initialise request-scoped store and response state
	requestStore := store.NewRequestStore()
	responseState := response.NewResponseState()

	// Handle system endpoints
	if handleSystemEndpoint(w, req) {
		return
	}

	// Process each plugin
	for _, plg := range plugins {
		// Standard response processor
		responseProc := response.NewProcessor(imposterConfig, plg.GetConfigDir())

		// Process request with handler
		plg.HandleRequest(req, requestStore, responseState, responseProc)

		// If the response has been handled by the handler, break the loop
		if responseState.Handled {
			break
		}
	}

	// If no handler handled the response, return 404
	if !responseState.Handled {
		handleNotFound(req, responseState, plugins)
	}

	// Add CORS headers to the response if configured in any plugin and Origin header is present
	if req.Header.Get("Origin") != "" {
		for _, plg := range plugins {
			if plg.GetConfig().Cors != nil {
				addCORSHeaders(w, req, plg.GetConfig().Cors)
				break
			}
		}
	}

	logger.Infof("handled request - method:%s, path:%s, status:%d, length:%d",
		req.Method, req.URL.Path, responseState.StatusCode, len(responseState.Body))

	// Write response to client
	responseState.WriteToResponseWriter(w)
}

// handleSystemEndpoint handles system-level endpoints like /system/store and /system/status
func handleSystemEndpoint(w http.ResponseWriter, r *http.Request) bool {
	switch {
	case strings.HasPrefix(r.URL.Path, "/system/store"):
		HandleStoreRequest(w, r)
		return true
	case r.URL.Path == "/system/status":
		handleStatusRequest(w, r)
		return true
	}
	return false
}
