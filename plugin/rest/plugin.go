package rest

import "github.com/imposter-project/imposter-go/internal/config"

// PluginHandler handles REST API requests
type PluginHandler struct {
	config         *config.Config
	configDir      string
	imposterConfig *config.ImposterConfig
}

// NewPluginHandler creates a new REST handler
func NewPluginHandler(cfg *config.Config, configDir string, imposterConfig *config.ImposterConfig) (*PluginHandler, error) {
	return &PluginHandler{
		config:         cfg,
		configDir:      configDir,
		imposterConfig: imposterConfig,
	}, nil
}

// GetConfigDir returns the original config directory
func (h *PluginHandler) GetConfigDir() string {
	return h.configDir
}

func (h *PluginHandler) GetConfig() *config.Config {
	return h.config
}
