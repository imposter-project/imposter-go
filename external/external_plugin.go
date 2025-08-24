package external

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

type ExternalPluginHandler struct {
	config         *config.Config
	imposterConfig *config.ImposterConfig
	loadedPlugin   *LoadedPlugin
}

// NewExternalPluginHandler creates a new external plugin handler
func NewExternalPluginHandler(loadedPlugin *LoadedPlugin, cfg *config.Config, imposterConfig *config.ImposterConfig) (*ExternalPluginHandler, error) {
	return &ExternalPluginHandler{
		config:         cfg,
		imposterConfig: imposterConfig,
		loadedPlugin:   loadedPlugin,
	}, nil
}

func (e ExternalPluginHandler) GetConfig() *config.Config {
	return e.config
}

func (e ExternalPluginHandler) HandleRequest(exch *exchange.Exchange, respProc response.Processor) {
	plg := e.loadedPlugin
	logger.Debugf("handling request with external plugin: %s", plg.Name)
	impl := *plg.impl
	args := ConvertToExternalRequest(exch)
	resp := impl.Handle(args)

	if resp.StatusCode == 0 {
		logger.Tracef("plugin %s did not handle the request", plg.Name)
		return
	}
	logger.Debugf("response from plugin %s: status=%d body=%d bytes", plg.Name, resp.StatusCode, len(resp.Body))
	ConvertFromExternalResponse(exch, &resp)
	respProc(exch, nil, &config.Response{})
}
