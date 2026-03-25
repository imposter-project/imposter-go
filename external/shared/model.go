package shared

import "net/url"

type HandlerRequest struct {
	Method  string
	Path    string
	Query   url.Values
	Headers map[string]string
	Body    []byte
}

type HandlerResponse struct {
	// If StatusCode is 0, the plugin did not handle the request.
	StatusCode int
	Headers    map[string]string
	Body       []byte
	File       string

	// FileBaseDir is the base directory from which the response file path should be resolved.
	FileBaseDir string

	// FileName is the name of the file, not its path. Used for MIME type detection.
	FileName string
}

type ServerConfig struct {
	URL string
}

type ExternalConfig struct {
	Server  ServerConfig
	Configs []LightweightConfig
}

type LightweightConfig struct {
	ConfigDir    string
	Plugin       string
	SpecFile     string
	WSDLFile     string
	PluginConfig []byte
}

// PluginCapabilities describes what capabilities a plugin supports.
type PluginCapabilities struct {
	// HandleRequests indicates the plugin can handle HTTP requests.
	HandleRequests bool

	// GenerateFakeData indicates the plugin can generate fake data.
	GenerateFakeData bool
}

// FakeDataRequest describes what fake data to generate.
type FakeDataRequest struct {
	// ExprCategory is the Datafaker-style category, e.g. "Name".
	// Used for template expressions like ${fake.Name.firstName}.
	ExprCategory string

	// ExprProperty is the Datafaker-style property, e.g. "firstName".
	ExprProperty string

	// PropertyName is an OpenAPI property name to infer fake data from, e.g. "firstName".
	PropertyName string

	// Format is an OpenAPI string format to infer fake data from, e.g. "email".
	Format string
}

// FakeDataResponse is the result of a fake data generation request.
type FakeDataResponse struct {
	// Value is the generated fake data value.
	Value string

	// Found indicates whether a generator matched the request.
	Found bool
}

// ExternalHandler defines the interface for external plugins to implement.
type ExternalHandler interface {
	// Configure is called to initialise the plugin with the loaded configuration.
	// It returns the plugin's capabilities.
	Configure(cfg ExternalConfig) (PluginCapabilities, error)

	// Handle processes the given request and returns a response.
	// If the response code is 0 or 404, the plugin did not handle the request.
	Handle(args HandlerRequest) HandlerResponse

	// GenerateFakeData generates fake data based on the request.
	// Only plugins with the GenerateFakeData capability should implement this.
	GenerateFakeData(req FakeDataRequest) FakeDataResponse
}
