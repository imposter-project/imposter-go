package handler

import (
	"net/http"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/plugin"
	"github.com/imposter-project/imposter-go/internal/rest"
	"github.com/imposter-project/imposter-go/internal/soap"
)

// PluginHandler defines the interface that all plugin handlers must implement
type PluginHandler interface {
	HandleRequest(r *http.Request) *plugin.ResponseState
}

// HandleRequest processes incoming HTTP requests and routes them to the appropriate handler
func HandleRequest(w http.ResponseWriter, r *http.Request, configDir string, configs []config.Config, imposterConfig *config.ImposterConfig) {
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
			handler, err = soap.NewHandler(&cfg, configDir)
		default:
			http.Error(w, "Unsupported plugin type", http.StatusInternalServerError)
			return
		}

		if err != nil {
			http.Error(w, "Failed to initialize handler", http.StatusInternalServerError)
			return
		}

		// Get response state from handler
		responseState := handler.HandleRequest(r)

		// Write response to client
		responseState.WriteToResponseWriter(w)
		return
	}
}

// handleSystemEndpoint handles system-level endpoints like /system/store
func handleSystemEndpoint(w http.ResponseWriter, r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, "/system/store") {
		HandleStoreRequest(w, r)
		return true
	}
	return false
}
