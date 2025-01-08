package adapter

import (
	"os"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/plugin"
)

// InitialiseImposter performs common initialisation tasks for all adapters
func InitialiseImposter(configDirArg string) (*config.ImposterConfig, string, []plugin.Plugin) {
	logger.Infoln("starting imposter-go...")

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
	plugins := plugin.LoadPlugins(configs, configDir, imposterConfig)

	store.InitStoreProvider()
	store.PreloadStores(configDir, configs)

	return imposterConfig, configDir, plugins
}
