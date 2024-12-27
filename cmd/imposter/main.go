package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gatehill/imposter-go/internal/config"
	"github.com/gatehill/imposter-go/internal/parser"
	"github.com/gatehill/imposter-go/internal/server"
)

func main() {
	fmt.Println("Starting Imposter-Go...")

	imposterConfig := config.LoadConfig()

	if len(os.Args) < 2 {
		panic("Config directory path must be provided as the first argument")
	}

	configDir := os.Args[1]
	if info, err := os.Stat(configDir); os.IsNotExist(err) || !info.IsDir() {
		panic("Specified path is not a valid directory")
	}

	var configs []config.Config

	scanRecursive := (os.Getenv("IMPOSTER_CONFIG_SCAN_RECURSIVE") == "true")

	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip subdirectories if not scanning recursively
		if info.IsDir() && info.Name() != filepath.Base(configDir) && !scanRecursive {
			return filepath.SkipDir
		}

		if !info.IsDir() && (strings.HasSuffix(info.Name(), "-config.json") || strings.HasSuffix(info.Name(), "-config.yaml") || strings.HasSuffix(info.Name(), "-config.yml")) {
			fmt.Printf("Loading config file: %s\n", path)
			fileConfig, err := parser.ParseConfig(path)
			if err != nil {
				return err
			}
			// Prefix 'File' properties if in a subdirectory
			baseDir := filepath.Dir(path)
			relDir, err := filepath.Rel(configDir, baseDir)
			if err != nil {
				return err
			}
			for i := range fileConfig.Resources {
				if fileConfig.Resources[i].Response.File != "" && relDir != "." {
					fileConfig.Resources[i].Response.File = filepath.Join(relDir, fileConfig.Resources[i].Response.File)
				}
			}
			configs = append(configs, *fileConfig)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	// Optional: check that at least one config is rest
	for _, cfg := range configs {
		if cfg.Plugin != "rest" {
			panic("Unsupported plugin type")
		}
	}

	// Initialize and start the server with multiple configs
	srv := server.NewServer(imposterConfig, configDir, configs)
	srv.Start()
}
