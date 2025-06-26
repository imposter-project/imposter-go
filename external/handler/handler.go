package handler

type HandlerRequest struct {
	Method  string
	Path    string
	Headers map[string]string
}

type HandlerResponse struct {
	ConfigDir  string
	StatusCode int
	Headers    map[string]string
	Body       []byte
	File       string
}

type LightweightConfig struct {
	ConfigDir string
	Plugin    string
	SpecFile  string
}

// ExternalHandler defines the interface for external plugins to implement.
type ExternalHandler interface {
	// Configure is called to initialise the plugin with the loaded configuration.
	Configure(configs []LightweightConfig) error

	// Handle processes the given request and returns a response.
	// If the response code is 0 or 404, the plugin did not handle the request.
	Handle(args HandlerRequest) HandlerResponse
}
