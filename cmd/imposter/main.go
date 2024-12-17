package main

import (
	"fmt"
	"github.com/gatehill/imposter-go/internal/parser"
	"github.com/gatehill/imposter-go/internal/server"
	"os"
	"path"
)

func main() {
	fmt.Println("Starting Imposter-Go...")

	// Load configuration
	wd, err := os.Getwd()
	yamlConfig := path.Join(wd, "config/imposter-config.yaml")

	configData, err := parser.ParseConfig(yamlConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse config: %v", err))
	}

	if configData.Plugin != "rest" {
		panic("Unsupported plugin type")
	}

	// Initialize and start the server with resources
	srv := server.NewServer(yamlConfig, configData.Resources)
	srv.Start()
}
