package main

import (
	"github.com/imposter-project/imposter-go/internal/adapter"
	"github.com/imposter-project/imposter-go/internal/adapter/awslambda"
	"github.com/imposter-project/imposter-go/internal/adapter/httpserver"
)

func main() {
	// Create and start the appropriate adapter based on runtime mode
	var runtimeAdapter adapter.Adapter
	if adapter.IsLambda() {
		runtimeAdapter = awslambda.NewAdapter()
	} else {
		runtimeAdapter = httpserver.NewAdapter()
	}
	runtimeAdapter.Start()
}
