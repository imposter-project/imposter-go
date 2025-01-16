package openapi

import (
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
)

func TestNewPluginHandler(t *testing.T) {
	cfg := &config.Config{
		Plugin: "openapi",
	}

	handler, err := NewPluginHandler(cfg, ".", nil)
	if err != nil {
		t.Errorf("Failed to create OpenAPI plugin handler: %v", err)
	}

	if handler == nil {
		t.Error("Expected non-nil handler")
	}

	if handler.GetConfig() != cfg {
		t.Error("Expected handler config to match input config")
	}
}
