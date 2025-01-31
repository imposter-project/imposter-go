package soap

import (
	"fmt"
	"path/filepath"

	"github.com/imposter-project/imposter-go/internal/config"
)

// PluginHandler handles SOAP requests based on WSDL configuration
type PluginHandler struct {
	config         *config.Config
	configDir      string
	wsdlParser     WSDLParser
	imposterConfig *config.ImposterConfig
}

// NewPluginHandler creates a new SOAP handler
func NewPluginHandler(cfg *config.Config, configDir string, imposterConfig *config.ImposterConfig) (*PluginHandler, error) {
	// If WSDLFile is not absolute, make it relative to configDir
	wsdlPath := cfg.WSDLFile
	if !filepath.IsAbs(wsdlPath) {
		wsdlPath = filepath.Join(configDir, wsdlPath)
	}

	parser, err := newWSDLParser(wsdlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WSDL: %w", err)
	}

	// Augment existing config with generated interceptors based on the WSDL
	if err := augmentConfigWithWSDL(cfg, parser); err != nil {
		return nil, fmt.Errorf("failed to augment config with WSDL: %w", err)
	}

	return &PluginHandler{
		config:         cfg,
		configDir:      configDir,
		wsdlParser:     parser,
		imposterConfig: imposterConfig,
	}, nil
}

// GetConfigDir returns the original config directory
func (h *PluginHandler) GetConfigDir() string {
	return h.configDir
}

func (h *PluginHandler) GetConfig() *config.Config {
	return h.config
}
