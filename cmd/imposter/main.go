package main

import (
	"fmt"
	"os"

	"github.com/gatehill/imposter-go/internal/config"
	"github.com/gatehill/imposter-go/internal/server"
	"github.com/gatehill/imposter-go/internal/store"
)

func main() {
	fmt.Println("Starting Imposter-Go...")

	imposterConfig := config.LoadImposterConfig()

	if len(os.Args) < 2 {
		panic("Config directory path must be provided as the first argument")
	}

	configDir := os.Args[1]
	if info, err := os.Stat(configDir); os.IsNotExist(err) || !info.IsDir() {
		panic("Specified path is not a valid directory")
	}

	configs := config.LoadConfig(configDir)

	store.InitStoreProvider()
	store.PreloadStores(configDir, configs)

	// Optional: check that at least one config is rest
	for _, cfg := range configs {
		if cfg.Plugin != "rest" {
			panic("Unsupported plugin type")
		}
	}

	// Initialize and start the server with multiple configs
	srv := server.NewServer(imposterConfig, configDir, configs)
	srv.Start(imposterConfig)
}
