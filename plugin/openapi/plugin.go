package openapi

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
)

// PluginHandler handles OpenAPI mock requests
type PluginHandler struct {
	config         *config.Config
	configDir      string
	openApiParser  *OpenAPIParser
	imposterConfig *config.ImposterConfig
}

// NewPluginHandler creates a new OpenAPI plugin handler
func NewPluginHandler(cfg *config.Config, configDir string, imposterConfig *config.ImposterConfig) (*PluginHandler, error) {
	// If SpecFile is not absolute, make it relative to configDir
	specFile := cfg.SpecFile
	if !filepath.IsAbs(specFile) {
		specFile = filepath.Join(configDir, specFile)
	}

	parser, err := newOpenAPIParser(specFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI: %w", err)
	}

	// Augment existing config with generated interceptors based on the OpenAPI spec
	if err := augmentConfigWithOpenApiSpec(cfg, *parser); err != nil {
		return nil, fmt.Errorf("failed to augment config with OpenAPI spec: %w", err)
	}

	return &PluginHandler{
		config:         cfg,
		configDir:      configDir,
		openApiParser:  parser,
		imposterConfig: imposterConfig,
	}, nil
}

// GetConfig returns the plugin configuration
func (h *PluginHandler) GetConfig() *config.Config {
	return h.config
}

// HandleRequest handles incoming HTTP requests
func (h *PluginHandler) HandleRequest(r *http.Request, requestStore store.Store, responseState *response.ResponseState) {
	// TODO: Implement OpenAPI request handling
	// This will include:
	// - Parsing and validating against OpenAPI spec
	// - Request matching based on paths and operations
	// - Response generation based on examples/schemas
}
