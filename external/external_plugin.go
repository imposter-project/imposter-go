package external

import (
	"github.com/imposter-project/imposter-go/external/shared"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/pipeline"
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

// HandleRequest processes a request through the unified external plugin pipeline:
//  1. NormaliseRequest — plugin decides if it handles this request and optionally transforms it
//  2. Core pipeline — matching, interceptors, capture, steps, response, templating
//  3. TransformResponse — plugin transforms the pipeline's response or generates one from scratch
func (e ExternalPluginHandler) HandleRequest(exch *exchange.Exchange, respProc response.Processor) {
	plg := e.loadedPlugin
	impl := *plg.impl

	args := ConvertToExternalRequest(exch)
	logger.Tracef("calling NormaliseRequest on external plugin %s for %s %s", plg.Name, args.Method, args.Path)

	// Phase 1: ask plugin to normalise the request
	normResp, err := impl.NormaliseRequest(args)
	if err != nil {
		logger.Errorf("plugin %s NormaliseRequest failed: %v", plg.Name, err)
		return
	}
	if normResp.Skip {
		logger.Tracef("plugin %s skipped the request", plg.Name)
		return
	}

	// Preserve the original body for TransformResponse
	originalBody := exch.Request.Body

	// Apply normalised body (e.g., gRPC decodes protobuf to JSON)
	if len(normResp.Body) > 0 {
		exch.Request.Body = normResp.Body
	}

	// Apply normalised headers
	if len(normResp.Headers) > 0 {
		for key, value := range normResp.Headers {
			exch.Request.Request.Header.Set(key, value)
		}
	}

	// Phase 2+3: run pipeline, then call TransformResponse on the result
	pipelineHandled := false
	wrappedRespProc := func(exch *exchange.Exchange, reqMatcher *config.RequestMatcher, resp *config.Response) {
		// Run standard response processing (file loading, templating, etc.)
		respProc(exch, reqMatcher, resp)
		pipelineHandled = true

		// Call TransformResponse on the pipeline result
		transformReq := shared.TransformRequest{
			Method:          args.Method,
			Path:            args.Path,
			Query:           args.Query,
			Headers:         args.Headers,
			Body:            originalBody,
			Handled:         true,
			StatusCode:      exch.ResponseState.StatusCode,
			ResponseHeaders: copyHeaders(exch.ResponseState.Headers),
			ResponseBody:    exch.ResponseState.Body,
			Metadata:        normResp.Metadata,
		}
		logger.Tracef("calling TransformResponse on plugin %s (pipeline handled)", plg.Name)
		result, err := impl.TransformResponse(transformReq)
		if err != nil {
			logger.Errorf("plugin %s TransformResponse failed: %v", plg.Name, err)
			return
		}
		applyTransformResult(result, exch.ResponseState)
	}

	pipeline.RunPipeline(e.config, e.imposterConfig, exch, wrappedRespProc, nil)

	// If the pipeline did not match any resource, still call TransformResponse
	// so plugins like OIDC can generate responses for unmatched requests
	if !pipelineHandled {
		transformReq := shared.TransformRequest{
			Method:   args.Method,
			Path:     args.Path,
			Query:    args.Query,
			Headers:  args.Headers,
			Body:     originalBody,
			Handled:  false,
			Metadata: normResp.Metadata,
		}
		logger.Tracef("calling TransformResponse on plugin %s (pipeline did not handle)", plg.Name)
		result, err := impl.TransformResponse(transformReq)
		if err != nil {
			logger.Errorf("plugin %s TransformResponse failed: %v", plg.Name, err)
			return
		}
		if result.StatusCode > 0 {
			applyTransformResult(result, exch.ResponseState)
			exch.ResponseState.Handled = true
		}
	}
}

// copyHeaders creates a shallow copy of a headers map.
func copyHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	cp := make(map[string]string, len(headers))
	for k, v := range headers {
		cp[k] = v
	}
	return cp
}

// applyTransformResult applies a TransformResponseResult to the ResponseState.
func applyTransformResult(result shared.TransformResponseResult, rs *exchange.ResponseState) {
	if result.StatusCode > 0 {
		rs.StatusCode = result.StatusCode
	}
	if result.Body != nil {
		rs.Body = result.Body
	}
	if result.Headers != nil {
		for key, value := range result.Headers {
			rs.Headers[key] = value
		}
	}
	// If the plugin provided a FileName hint and did not set Content-Type
	// explicitly, infer it from the file extension.
	if result.FileName != "" {
		response.SetContentTypeHeader(rs, result.FileName, "", "")
	}
}
