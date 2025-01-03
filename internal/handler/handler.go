package handler

import (
	"net/http"
	"strings"

	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/plugin/rest"
	"github.com/imposter-project/imposter-go/plugin/soap"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
)

// PluginHandler defines the interface that all plugin handlers must implement
type PluginHandler interface {
	HandleRequest(r *http.Request, requestStore store.Store, responseState *response.ResponseState)
}

// HandleRequest processes incoming HTTP requests and routes them to the appropriate handler
func HandleRequest(w http.ResponseWriter, r *http.Request, configDir string, configs []config.Config, imposterConfig *config.ImposterConfig) {
	// Initialise request-scoped store and response state
	requestStore := make(store.Store)
	responseState := response.NewResponseState()

	// Handle system endpoints
	if handleSystemEndpoint(w, r) {
		return
	}

	// Process each config
	for _, cfg := range configs {
		var handler PluginHandler
		var err error

		switch cfg.Plugin {
		case "rest":
			handler, err = rest.NewHandler(&cfg, configDir, imposterConfig)
		case "soap":
			handler, err = soap.NewHandler(&cfg, configDir, imposterConfig)
		default:
			http.Error(w, "Unsupported plugin type", http.StatusInternalServerError)
			return
		}

		if err != nil {
			http.Error(w, "Failed to initialise handler", http.StatusInternalServerError)
			return
		}

		// Process request with handler
		handler.HandleRequest(r, requestStore, responseState)

		// If the response has been handled by the handler, break the loop
		if responseState.Handled {
			break
		}
	}

	// If no handler handled the response, return 404
	if !responseState.Handled {
		handleNotFound(r, responseState, configs)
	}

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
