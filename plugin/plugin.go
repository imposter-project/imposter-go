package plugin

import (
	"fmt"
	"net/http"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/plugin/openapi"
	"github.com/imposter-project/imposter-go/plugin/rest"
	"github.com/imposter-project/imposter-go/plugin/soap"
)

type Plugin interface {
	GetConfig() *config.Config
	HandleRequest(r *http.Request, requestStore store.Store, responseState *response.ResponseState)
}

// LoadPlugins loads plugins from the provided configs
func LoadPlugins(configs []config.Config, configDir string, imposterConfig *config.ImposterConfig) []Plugin {
	var plugins []Plugin

	// Process each config
	for _, cfg := range configs {
		var err error
		var plugin Plugin

		switch cfg.Plugin {
		case "openapi":
			plugin, err = openapi.NewPluginHandler(&cfg, configDir, imposterConfig)
		case "rest":
			plugin, err = rest.NewPluginHandler(&cfg, configDir, imposterConfig)
		case "soap":
			plugin, err = soap.NewPluginHandler(&cfg, configDir, imposterConfig)
		default:
			panic("Unsupported plugin type: " + cfg.Plugin)
		}

		if err != nil {
			panic(fmt.Errorf("failed to initialise plugin: %w", err))
		}
		plugins = append(plugins, plugin)
	}
	return plugins
}
