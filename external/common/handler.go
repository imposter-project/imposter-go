package common

type HandlerArgs struct {
	Method  string
	Path    string
	Headers map[string]string
}

// ExternalHandler defines the interface for external plugins to implement.
type ExternalHandler interface {
	// Handle processes the given path and returns a string response.
	Handle(args HandlerArgs) []byte
}
