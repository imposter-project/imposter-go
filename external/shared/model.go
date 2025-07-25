package shared

type HandlerRequest struct {
	Method  string
	Path    string
	Headers map[string]string
}

type HandlerResponse struct {
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
	ConfigDir string
	Plugin    string
	SpecFile  string
}

// ExternalHandler defines the interface for external plugins to implement.
type ExternalHandler interface {
	// Configure is called to initialise the plugin with the loaded configuration.
	Configure(cfg ExternalConfig) error

	// Handle processes the given request and returns a response.
	// If the response code is 0 or 404, the plugin did not handle the request.
	Handle(args HandlerRequest) HandlerResponse
}
