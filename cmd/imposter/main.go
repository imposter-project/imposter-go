package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/imposter-project/imposter-go/external"
	"github.com/imposter-project/imposter-go/internal/adapter"
	"github.com/imposter-project/imposter-go/internal/adapter/awslambda"
	"github.com/imposter-project/imposter-go/internal/adapter/httpserver"
	"github.com/imposter-project/imposter-go/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		// this is the behaviour and format expected by imposter-cli
		fmt.Println(version.Version)
		os.Exit(0)
	}

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
