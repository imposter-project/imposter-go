package handler

type HandlerRequest struct {
	Method  string
	Path    string
	Headers map[string]string
}

type HandlerResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

// ExternalHandler defines the interface for external plugins to implement.
type ExternalHandler interface {
	// Handle processes the given path and returns a string response.
	Handle(args HandlerRequest) HandlerResponse
}
