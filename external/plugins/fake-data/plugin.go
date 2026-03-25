package main

import (
	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
)

// FakeDataPlugin implements the ExternalHandler interface, providing
// fake data generation capabilities.
type FakeDataPlugin struct {
	logger hclog.Logger
}

func (f *FakeDataPlugin) Configure(_ shared.ExternalConfig) (shared.PluginCapabilities, error) {
	f.logger.Info("fake-data plugin configured")
	return shared.PluginCapabilities{GenerateFakeData: true}, nil
}

func (f *FakeDataPlugin) Handle(_ shared.HandlerRequest) shared.HandlerResponse {
	// This plugin does not handle HTTP requests.
	return shared.HandlerResponse{StatusCode: 0}
}

func (f *FakeDataPlugin) GenerateFakeData(req shared.FakeDataRequest) shared.FakeDataResponse {
	// Try expression-based generation first (${fake.Category.property})
	if req.ExprCategory != "" && req.ExprProperty != "" {
		if val, ok := Generate(req.ExprCategory, req.ExprProperty); ok {
			return shared.FakeDataResponse{Value: val, Found: true}
		}
	}

	// Try property name inference (OpenAPI property name → fake data)
	if req.PropertyName != "" {
		if val, ok := GenerateForPropertyName(req.PropertyName); ok {
			return shared.FakeDataResponse{Value: val, Found: true}
		}
	}

	// Try format inference (OpenAPI string format → fake data)
	if req.Format != "" {
		if val, ok := GenerateForFormat(req.Format); ok {
			return shared.FakeDataResponse{Value: val, Found: true}
		}
	}

	return shared.FakeDataResponse{}
}
