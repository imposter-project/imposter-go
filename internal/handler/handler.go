package handler

import (
	"github.com/imposter-project/imposter-go/internal/logger"
	"net/http"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/plugin"
)

// HandleRequest processes incoming HTTP requests and routes them to the appropriate handler
func HandleRequest(w http.ResponseWriter, req *http.Request, configDir string, plugins []plugin.Plugin, imposterConfig *config.ImposterConfig) {
	// Initialise request-scoped store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle system endpoints
	if handleSystemEndpoint(w, req) {
		return
	}

	// Process each config
	for _, plg := range plugins {
		// Process request with handler
		plg.HandleRequest(req, requestStore, responseState)

		// If the response has been handled by the handler, break the loop
		if responseState.Handled {
			break
		}
	}

	// If no handler handled the response, return 404
	if !responseState.Handled {
		handleNotFound(req, responseState, plugins)
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
