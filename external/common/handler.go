package common

// ExternalHandler defines the interface for external plugins to implement.
type ExternalHandler interface {
	// Handle processes the given path and returns a string response.
	Handle(path string) string
}
