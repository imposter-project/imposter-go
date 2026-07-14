package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// PluginHandler handles WebSocket mock connections
type PluginHandler struct {
	config         *config.Config
	imposterConfig *config.ImposterConfig
	respProc       response.Processor
	upgrader       websocket.Upgrader
}

// NewPluginHandler creates a new WebSocket handler
func NewPluginHandler(cfg *config.Config, imposterConfig *config.ImposterConfig) (*PluginHandler, error) {
	h := &PluginHandler{
		config:         cfg,
		imposterConfig: imposterConfig,
		// Connections outlive the HTTP exchange, so the plugin owns a response
		// processor rather than retaining the per-request one.
		respProc: response.NewProcessor(imposterConfig, cfg.ConfigDir),
		upgrader: websocket.Upgrader{
			// Mock servers accept connections from any origin
			CheckOrigin: func(r *http.Request) bool { return true },
			// Suppress gorilla's direct error writes; handshake errors are
			// recorded in the buffered ResponseState instead.
			Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {},
		},
	}

	if imposterConfig.HTTP2Enabled && imposterConfig.TLSEnabled() {
		logger.Debugf("websocket upgrades require HTTP/1.1; HTTP/2 streams cannot be upgraded")
	}

	return h, nil
}

func (h *PluginHandler) GetConfig() *config.Config {
	return h.config
}
