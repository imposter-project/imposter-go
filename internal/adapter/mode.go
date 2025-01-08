package adapter

import (
	"os"
	"sync"
)

// Mode represents the runtime mode of the application
type Mode int

const (
	ModeUnknown Mode = iota
	ModeLambda
	ModeHTTPServer
)

var (
	currentMode Mode
	modeOnce    sync.Once
)

// init determines the runtime mode during package initialization
func init() {
	DetectMode()
}

// DetectMode determines and sets the runtime mode of the application
func DetectMode() Mode {
	modeOnce.Do(func() {
		if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
			currentMode = ModeLambda
		} else {
			currentMode = ModeHTTPServer
		}
	})
	return currentMode
}

// IsLambda returns true if running in AWS Lambda mode
func IsLambda() bool {
	return currentMode == ModeLambda
}

// IsHTTPServer returns true if running in HTTP server mode
func IsHTTPServer() bool {
	return currentMode == ModeHTTPServer
}
