package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gatehill/imposter-go/internal/parser"
	"github.com/gatehill/imposter-go/internal/server"
)

func main() {
	fmt.Println("Starting Imposter-Go...")

	if len(os.Args) < 2 {
		panic("Config directory path must be provided as the first argument")
	}

	configDir := os.Args[1]
	if info, err := os.Stat(configDir); os.IsNotExist(err) || !info.IsDir() {
		panic("Specified path is not a valid directory")
	}

	var combinedConfig parser.Config

	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), "-config.json") || strings.HasSuffix(info.Name(), "-config.yaml") || strings.HasSuffix(info.Name(), "-config.yml")) {
			fileConfig, err := parser.ParseConfig(path)
			if err != nil {
				return err
			}
			if combinedConfig.Plugin == "" {
				combinedConfig.Plugin = fileConfig.Plugin
			} else if combinedConfig.Plugin != fileConfig.Plugin {
				panic("Mismatched plugin types encountered")
			}

			// Merge resources
			combinedConfig.Resources = append(combinedConfig.Resources, fileConfig.Resources...)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	if combinedConfig.Plugin != "rest" {
		panic("Unsupported plugin type")
	}

	// Initialize and start the server with resources
	srv := server.NewServer(configDir, combinedConfig.Resources)
	srv.Start()
}
