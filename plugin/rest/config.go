package rest

import "github.com/imposter-project/imposter-go/internal/config"

func (h *PluginHandler) GetConfig() *config.Config {
	return h.config
}
