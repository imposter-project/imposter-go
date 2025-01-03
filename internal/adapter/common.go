package adapter

import (
	"fmt"
	"os"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
)

// InitialiseImposter performs common initialisation tasks for all adapters
func InitialiseImposter(configDirArg string) (*config.ImposterConfig, string, []config.Config) {
	fmt.Println("Starting Imposter-Go...")

	imposterConfig := config.LoadImposterConfig()

	var configDir string
	if configDirArg != "" {
		configDir = configDirArg
	} else {
		configDir = os.Getenv("IMPOSTER_CONFIG_DIR")
		if configDir == "" {
			panic("Config directory path must be provided either as an argument or via IMPOSTER_CONFIG_DIR environment variable")
		}
	}

	if info, err := os.Stat(configDir); os.IsNotExist(err) || !info.IsDir() {
		panic("Specified path is not a valid directory")
	}

	configs := config.LoadConfig(configDir)

	store.InitStoreProvider()
	store.PreloadStores(configDir, configs)

	return imposterConfig, configDir, configs
}
