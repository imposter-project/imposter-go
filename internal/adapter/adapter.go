package adapter

// Adapter represents a runtime adapter for the application
type Adapter interface {
	// Start begins the adapter's runtime execution. It returns an error if
	// startup fails (for example, invalid configuration) so the caller can
	// report it cleanly and exit with a non-zero status.
	Start() error
}
