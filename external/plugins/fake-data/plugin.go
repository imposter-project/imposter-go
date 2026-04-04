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

func (f *FakeDataPlugin) NormaliseRequest(_ shared.HandlerRequest) (shared.NormaliseResponse, error) {
	return shared.NormaliseResponse{Skip: true}, nil
}

func (f *FakeDataPlugin) TransformResponse(_ shared.TransformRequest) (shared.TransformResponseResult, error) {
	return shared.TransformResponseResult{}, nil
}

func (f *FakeDataPlugin) GenerateFakeData(req shared.FakeDataRequest) (shared.FakeDataResponse, error) {
	// Try expression-based generation first (${fake.Category.property})
	if req.ExprCategory != "" && req.ExprProperty != "" {
		if val, ok := Generate(req.ExprCategory, req.ExprProperty); ok {
			return shared.FakeDataResponse{Value: val, Found: true}, nil
		}
	}

	// Try property name inference (OpenAPI property name → fake data)
	if req.PropertyName != "" {
		if val, ok := GenerateForPropertyName(req.PropertyName); ok {
			return shared.FakeDataResponse{Value: val, Found: true}, nil
		}
	}

	// Try format inference (OpenAPI string format → fake data)
	if req.Format != "" {
		if val, ok := GenerateForFormat(req.Format); ok {
			return shared.FakeDataResponse{Value: val, Found: true}, nil
		}
	}

	return shared.FakeDataResponse{}, nil
}
