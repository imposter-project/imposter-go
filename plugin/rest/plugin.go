package rest

import (
	"github.com/imposter-project/imposter-go/internal/config"
)

// PluginHandler handles REST API requests
type PluginHandler struct {
	config         *config.Config
	imposterConfig *config.ImposterConfig
}

// NewPluginHandler creates a new REST handler
func NewPluginHandler(cfg *config.Config, imposterConfig *config.ImposterConfig) (*PluginHandler, error) {
	return &PluginHandler{
		config:         cfg,
		imposterConfig: imposterConfig,
	}, nil
}

func (h *PluginHandler) GetConfig() *config.Config {
	return h.config
}
