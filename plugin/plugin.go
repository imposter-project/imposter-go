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
	// GetConfig returns the plugin configuration
	GetConfig() *config.Config

	// HandleRequest processes incoming HTTP requests and routes them to the appropriate handler
	HandleRequest(exch *exchange.Exchange, respProc response.Processor)
}

// LoadPlugin loads the plugin from the provided config
func LoadPlugin(cfg *config.Config, imposterConfig *config.ImposterConfig) Plugin {
	var err error
	var plg Plugin

	switch cfg.Plugin {
	case "openapi":
		plg, err = openapi.NewPluginHandler(cfg, imposterConfig)
	case "rest":
		plg, err = rest.NewPluginHandler(cfg, imposterConfig)
	case "soap":
		plg, err = soap.NewPluginHandler(cfg, imposterConfig)
	default:
		panic("Unsupported plugin type: " + cfg.Plugin)
	}

	if err != nil {
		panic(fmt.Errorf("failed to initialise plugin: %w", err))
	}

	return plg
}

// LoadPlugins loads multiple plugins based on the provided configurations
// and returns a slice of Plugin interfaces.
func LoadPlugins(configs []config.Config, imposterConfig *config.ImposterConfig) []Plugin {
	var plugins []Plugin
	for _, cfg := range configs {
		plugins = append(plugins, LoadPlugin(&cfg, imposterConfig))
	}
	return plugins
}
