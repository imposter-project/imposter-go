package main

import (
	"github.com/imposter-project/imposter-go/external"
	"github.com/imposter-project/imposter-go/internal/adapter"
	"github.com/imposter-project/imposter-go/internal/adapter/awslambda"
	"github.com/imposter-project/imposter-go/internal/adapter/httpserver"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cleanup()
		os.Exit(0)
	}()

	// Create and start the appropriate adapter based on runtime mode
	var runtimeAdapter adapter.Adapter
	if adapter.IsLambda() {
		runtimeAdapter = awslambda.NewAdapter()
	} else {
		runtimeAdapter = httpserver.NewAdapter()
	}
	runtimeAdapter.Start()
	cleanup()
}

func cleanup() {
	external.StopExternalPlugins()
}
