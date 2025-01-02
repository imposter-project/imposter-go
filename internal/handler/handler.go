package handler

import (
	"net/http"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/rest"
	"github.com/imposter-project/imposter-go/internal/soap"
)

// HandleRequest processes incoming HTTP requests and routes them to the appropriate handler
func HandleRequest(w http.ResponseWriter, r *http.Request, configDir string, configs []config.Config, imposterConfig *config.ImposterConfig) {
	// Handle system endpoints
	if handleSystemEndpoint(w, r) {
		return
	}

	// Process each config
	for _, cfg := range configs {
		switch cfg.Plugin {
		case "rest":
			handleRestRequest(w, r, &cfg, configDir, imposterConfig)
		case "soap":
			handleSOAPRequest(w, r, &cfg, configDir)
		default:
			http.Error(w, "Unsupported plugin type", http.StatusInternalServerError)
			return
		}
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

// handleRestRequest handles REST API requests
func handleRestRequest(w http.ResponseWriter, r *http.Request, cfg *config.Config, configDir string, imposterConfig *config.ImposterConfig) {
	handler, err := rest.NewHandler(cfg, configDir, imposterConfig)
	if err != nil {
		http.Error(w, "Failed to initialize REST handler", http.StatusInternalServerError)
		return
	}
	handler.HandleRequest(w, r)
}

// handleSOAPRequest handles SOAP requests using the SOAP plugin
func handleSOAPRequest(w http.ResponseWriter, r *http.Request, cfg *config.Config, configDir string) {
	handler, err := soap.NewHandler(cfg, configDir)
	if err != nil {
		http.Error(w, "Failed to initialize SOAP handler", http.StatusInternalServerError)
		return
	}
	handler.HandleRequest(w, r)
}
