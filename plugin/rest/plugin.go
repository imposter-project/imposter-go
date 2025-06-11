package rest

import (
	"fmt"
	"os"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
)

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

// getStoreProvider returns the global store provider
func (h *PluginHandler) getStoreProvider() store.StoreProvider {
	return store.GetStoreProvider()
}

// getInstanceID generates a unique instance ID for this server instance
func (h *PluginHandler) getInstanceID() string {
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s-%d-%d", hostname, pid, timestamp)
}
