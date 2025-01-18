package adapter

import (
	"os"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/plugin"
)

// InitialiseImposter performs common initialisation tasks for all adapters
func InitialiseImposter(configDirArg string) (*config.ImposterConfig, []plugin.Plugin) {
	logger.Infoln("starting imposter-go...")

	imposterConfig := config.LoadImposterConfig()
	configDirs := getConfigDirs(configDirArg)

	store.InitStoreProvider()

	var plugins []plugin.Plugin
	for _, configDir := range configDirs {
		if info, err := os.Stat(configDir); os.IsNotExist(err) || !info.IsDir() {
			panic("Specified path is not a valid directory")
		}

		cfgs := config.LoadConfig(configDir)
		plgs := plugin.LoadPlugins(cfgs, configDir, imposterConfig)

		store.PreloadStores(configDir, cfgs)

		plugins = append(plugins, plgs...)
	}
	return imposterConfig, plugins
}

func getConfigDirs(configDirArg string) []string {
	var configDirRaw string
	if configDirArg != "" {
		configDirRaw = configDirArg
	} else {
		configDirRaw = os.Getenv("IMPOSTER_CONFIG_DIR")
		if configDirRaw == "" {
			panic("Config directory path must be provided either as an argument or via IMPOSTER_CONFIG_DIR environment variable")
		}
	}
	configDirs := strings.Split(configDirRaw, ",")
	return configDirs
}
