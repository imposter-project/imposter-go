package external

import (
	"net/http"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/steps"
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
	r := exch.Request.Request
	responseState := exch.ResponseState

	// Get system XML namespaces
	var systemNamespaces map[string]string
	if e.config.System != nil {
		systemNamespaces = e.config.System.XMLNamespaces
	}

	// Process interceptors first
	for _, interceptorCfg := range e.config.Interceptors {
		score, _ := matcher.CalculateMatchScore(exch, &interceptorCfg.RequestMatcher, systemNamespaces, e.imposterConfig)
		if score > 0 {
			logger.Infof("matched interceptor - method:%s, path:%s", r.Method, r.URL.Path)
			if interceptorCfg.Capture != nil {
				capture.CaptureRequestData(e.imposterConfig, &interceptorCfg.RequestMatcher, interceptorCfg.Capture, exch)
			}

			// Execute steps if present
			if len(interceptorCfg.Steps) > 0 {
				if err := steps.RunSteps(interceptorCfg.Steps, exch, e.imposterConfig, e.config.ConfigDir, responseState, &interceptorCfg.RequestMatcher); err != nil {
					logger.Errorf("failed to execute interceptor steps: %v", err)
					responseState.StatusCode = http.StatusInternalServerError
					responseState.Body = []byte("Failed to execute steps")
					responseState.Handled = true
					return
				}
				if responseState.Handled {
					return
				}
			}

			if interceptorCfg.Response != nil {
				respProc(exch, &interceptorCfg.RequestMatcher, interceptorCfg.Response)
			}
			if !interceptorCfg.Continue {
				responseState.HandledWithResource(&interceptorCfg.BaseResource)
				return
			}
		}
	}

	plg := e.loadedPlugin
	impl := *plg.impl
	args := ConvertToExternalRequest(exch)
	logger.Tracef("checking if external plugin %s can handle request %s %s", plg.Name, args.Method, args.Path)
	resp := impl.Handle(args)

	if resp.StatusCode == 0 {
		logger.Tracef("plugin %s did not handle the request", plg.Name)
		return
	}
	logger.Debugf("response from plugin %s: status=%d body=%d bytes", plg.Name, resp.StatusCode, len(resp.Body))
	ConvertFromExternalResponse(exch, &resp)
	respProc(exch, nil, &config.Response{})
}
