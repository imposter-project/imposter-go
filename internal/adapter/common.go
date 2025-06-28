package adapter

import (
	"github.com/imposter-project/imposter-go/external"
	"github.com/imposter-project/imposter-go/internal/version"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"os"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/plugin"
)

// InitialiseImposter performs common initialisation tasks for all adapters
func InitialiseImposter(configDirArg string) (*config.ImposterConfig, []plugin.Plugin) {
	logger.Infof("starting imposter-go %s...", version.Version)

	imposterConfig := config.LoadImposterConfig()
	configDirs := getConfigDirs(configDirArg)

	store.InitStoreProvider()

	var plugins []plugin.Plugin
	totalConfigs := 0

	for _, configDir := range configDirs {
		if info, err := os.Stat(configDir); os.IsNotExist(err) || !info.IsDir() {
			panic("Specified path is not a valid directory")
		}

		cfgs := config.LoadConfig(configDir, imposterConfig)
		totalConfigs += len(cfgs)
		plgs := plugin.LoadPlugins(cfgs, configDir, imposterConfig)

		store.PreloadStores(configDir, cfgs)

		plugins = append(plugins, plgs...)
	}

	// Exit if no configuration files were found
	if totalConfigs == 0 {
		logger.Errorf("no configuration files found in specified directories: %v", configDirs)
		os.Exit(1)
	}

	// Pre-calculate resource IDs for all loaded configurations.
	// Note: we retrieve the config from each plugin to ensure it includes
	// any dynamic resources added by the plugin.
	for _, plg := range plugins {
		config.PreCalculateResourceID(plg.GetConfig())
	}

	if err := external.StartExternalPlugins(plugins); err != nil {
		panic(err)
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
