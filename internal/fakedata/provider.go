package fakedata

import (
	"sync"

	"github.com/imposter-project/imposter-go/pkg/logger"
)

// Request describes what fake data to generate.
type Request struct {
	// ExprCategory is the Datafaker-style category, e.g. "Name".
	ExprCategory string

	// ExprProperty is the Datafaker-style property, e.g. "firstName".
	ExprProperty string

	// PropertyName is an OpenAPI property name to infer fake data from.
	PropertyName string

	// Format is an OpenAPI string format to infer fake data from.
	Format string
}

// Response is the result of a fake data generation request.
type Response struct {
	Value string
	Found bool
}

// Provider generates fake data via an external plugin.
type Provider interface {
	GenerateFakeData(req Request) Response
}

var (
	providerMu sync.RWMutex
	provider   Provider
)

// RegisterProvider sets the global fake data provider.
func RegisterProvider(p Provider) {
	providerMu.Lock()
	defer providerMu.Unlock()
	provider = p
}

// GetProvider returns the registered fake data provider, or nil.
func GetProvider() Provider {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return provider
}

// Generate calls the registered provider to generate fake data for a
// Datafaker-style category and property (e.g. "Name", "firstName").
// Returns the generated value or empty string if no provider is registered.
func Generate(category, property string) string {
	p := GetProvider()
	if p == nil {
		logger.Tracef("no fake data provider registered for ${fake.%s.%s}", category, property)
		return ""
	}
	resp := p.GenerateFakeData(Request{
		ExprCategory: category,
		ExprProperty: property,
	})
	if resp.Found {
		return resp.Value
	}
	return ""
}

// GenerateForPropertyName asks the provider to infer fake data from an
// OpenAPI property name (e.g. "firstName" → a fake first name).
func GenerateForPropertyName(propertyName string) (string, bool) {
	p := GetProvider()
	if p == nil {
		return "", false
	}
	resp := p.GenerateFakeData(Request{
		PropertyName: propertyName,
	})
	return resp.Value, resp.Found
}

// GenerateForFormat asks the provider to infer fake data from an OpenAPI
// string format (e.g. "email" → a fake email address).
func GenerateForFormat(format string) (string, bool) {
	p := GetProvider()
	if p == nil {
		return "", false
	}
	resp := p.GenerateFakeData(Request{
		Format: format,
	})
	return resp.Value, resp.Found
}
