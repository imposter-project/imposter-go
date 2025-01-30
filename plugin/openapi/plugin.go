package openapi

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/plugin/rest"
	"net/http"
	"path/filepath"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
)

// PluginHandler handles OpenAPI mock requests
type PluginHandler struct {
	config            *config.Config
	configDir         string
	openApiParser     OpenAPIParser
	imposterConfig    *config.ImposterConfig
	restPluginHandler *rest.PluginHandler
}

// NewPluginHandler creates a new OpenAPI plugin handler
func NewPluginHandler(cfg *config.Config, configDir string, imposterConfig *config.ImposterConfig) (*PluginHandler, error) {
	// If SpecFile is not absolute, make it relative to configDir
	specFile := cfg.SpecFile
	if !filepath.IsAbs(specFile) {
		specFile = filepath.Join(configDir, specFile)
	}

	opts := parserOptions{
		stripServerPath: cfg.StripServerPath,
	}
	parser, err := newOpenAPIParser(specFile, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI: %w", err)
	}

	// Augment existing config with generated interceptors based on the OpenAPI spec
	if err := augmentConfigWithOpenApiSpec(cfg, parser); err != nil {
		return nil, fmt.Errorf("failed to augment config with OpenAPI spec: %w", err)
	}

	restPluginHandler, err := rest.NewPluginHandler(cfg, configDir, imposterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST plugin handler: %w", err)
	}

	return &PluginHandler{
		config:            cfg,
		configDir:         configDir,
		openApiParser:     parser,
		imposterConfig:    imposterConfig,
		restPluginHandler: restPluginHandler,
	}, nil
}

// GetConfig returns the plugin configuration
func (h *PluginHandler) GetConfig() *config.Config {
	return h.config
}

// HandleRequest handles incoming HTTP requests
func (h *PluginHandler) HandleRequest(
	r *http.Request,
	requestStore *store.Store,
	responseState *response.ResponseState,
	respPreprocessor response.Processor,
) {
	// TODO validate request against OpenAPI spec

	wrapped := func(reqMatcher *config.RequestMatcher, rs *response.ResponseState, r *http.Request, resp *config.Response, requestStore *store.Store) {
		h.preprocessResponse(reqMatcher, rs, r, resp, requestStore, respPreprocessor)
	}

	h.restPluginHandler.HandleRequest(r, requestStore, responseState, wrapped)
}

func (h *PluginHandler) getMatchedSpecResponse(requestStore store.Store, r *http.Request) *Response {
	openApiOpId := requestStore["_matched-openapi-operation"]
	if openApiOpId == nil {
		logger.Tracef("no OpenAPI operation matched for request %s %s", r.Method, r.URL.Path)
		return nil
	}
	op := h.openApiParser.GetOperation(openApiOpId.(string))

	openApiRespId := requestStore["_matched-openapi-response"]
	if openApiRespId == nil {
		// if an operation is matched, a response should also be matched
		logger.Errorf("no OpenAPI response matched for request %s %s", r.Method, r.URL.Path)
		return nil
	}

	var openApiResp *Response
	for _, oar := range op.Responses {
		for _, statusResp := range oar {
			if statusResp.UniqueID == openApiRespId.(string) {
				openApiResp = &statusResp
				break
			}
		}
	}
	return openApiResp
}
