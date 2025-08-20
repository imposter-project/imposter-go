package plugin

import (
	"fmt"
	"github.com/imposter-project/imposter-go/external"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/pkg/logger"

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

// LoadPlugins loads multiple plugins based on the provided configurations
// and returns a slice of Plugin interfaces.
func LoadPlugins(configs []config.Config, imposterConfig *config.ImposterConfig, externalPlugins []external.LoadedPlugin) ([]Plugin, error) {
	var plugins []Plugin
	for _, cfg := range configs {
		plg, err := loadPlugin(&cfg, imposterConfig, externalPlugins)
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin '%s': %w", cfg.Plugin, err)
		}
		plugins = append(plugins, *plg)
	}
	return plugins, nil
}

// loadPlugin loads the plugin from the provided config
func loadPlugin(
	cfg *config.Config,
	imposterConfig *config.ImposterConfig,
	externalPlugins []external.LoadedPlugin,
) (*Plugin, error) {
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
		for _, e := range externalPlugins {
			if e.Name == cfg.Plugin {
				logger.Tracef("found external plugin: %s", e.Name)
				plg, err = external.NewExternalPluginHandler(&e, cfg, imposterConfig)
				break
			}
		}
	}
	if plg == nil {
		err = fmt.Errorf("plugin '%s' not found", cfg.Plugin)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialise plugin: %w", err)
	}
	return &plg, nil
}
