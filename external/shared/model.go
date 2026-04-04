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
	// HandleRequests indicates the plugin can handle HTTP requests
	// via the NormaliseRequest/TransformResponse pipeline.
	HandleRequests bool

	// GenerateFakeData indicates the plugin can generate fake data.
	GenerateFakeData bool
}

// NormaliseResponse is returned by NormaliseRequest.
type NormaliseResponse struct {
	// Skip means the plugin does not handle this request at all.
	// When true, the pipeline is not run and TransformResponse is not called.
	Skip bool

	// Body is the normalised request body for the pipeline.
	// For example, a gRPC plugin decodes protobuf to JSON here.
	// If empty, the original request body is used unchanged.
	Body []byte

	// Headers to set or override on the request before the pipeline runs.
	Headers map[string]string

	// Metadata is opaque data passed through to TransformResponse.
	// For example, a gRPC plugin stores which proto method descriptor
	// to use for encoding the response.
	Metadata []byte
}

// TransformRequest is passed to TransformResponse after the pipeline runs.
type TransformRequest struct {
	// Original request context
	Method  string
	Path    string
	Query   url.Values
	Headers map[string]string
	Body    []byte // original (pre-normalised) request body

	// Pipeline result
	Handled         bool              // whether the pipeline matched a resource
	StatusCode      int               // pipeline response status code
	ResponseHeaders map[string]string // pipeline response headers
	ResponseBody    []byte            // pipeline response body

	// Metadata from NormaliseResponse
	Metadata []byte
}

// TransformResponseResult is returned by the TransformResponse method.
// If the pipeline handled the request, this transforms the response.
// If the pipeline did not handle it, this can generate a response from scratch.
type TransformResponseResult struct {
	// StatusCode is the HTTP status code for the response.
	// If 0, the plugin did not produce a response (request falls through).
	StatusCode int
	Headers    map[string]string
	Body       []byte
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

	// NormaliseRequest is called before the core pipeline runs.
	// The plugin can indicate whether it handles this request (Skip=false)
	// and optionally transform the request body/headers for the pipeline.
	NormaliseRequest(args HandlerRequest) (NormaliseResponse, error)

	// TransformResponse is called after the core pipeline runs (or when
	// the pipeline found no matching resource). The plugin can transform
	// the pipeline's response or generate a response from scratch.
	TransformResponse(args TransformRequest) (TransformResponseResult, error)

	// GenerateFakeData generates fake data based on the request.
	// Only plugins with the GenerateFakeData capability should implement this.
	GenerateFakeData(req FakeDataRequest) (FakeDataResponse, error)
}
