package plugin

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/exchange"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/plugin/openapi"
	"github.com/imposter-project/imposter-go/plugin/rest"
	"github.com/imposter-project/imposter-go/plugin/soap"
)

type Plugin interface {
	// GetConfigDir returns the original config directory, *which might be a parent*,
	// from which the config file was discovered.
	GetConfigDir() string

	// GetConfig returns the plugin configuration
	GetConfig() *config.Config

	// HandleRequest processes incoming HTTP requests and routes them to the appropriate handler
	HandleRequest(exch *exchange.Exchange, respProc response.Processor)
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
