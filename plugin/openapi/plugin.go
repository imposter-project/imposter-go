package openapi

import (
	"fmt"
	"github.com/imposter-project/imposter-go/plugin/rest"
	"path/filepath"

	"github.com/imposter-project/imposter-go/internal/config"
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

// GetConfigDir returns the original config directory
func (h *PluginHandler) GetConfigDir() string {
	return h.configDir
}

// GetConfig returns the plugin configuration
func (h *PluginHandler) GetConfig() *config.Config {
	return h.config
}
