package adapter

import (
	"fmt"
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

	var configs []config.Config
	for _, configDir := range configDirs {
		if info, err := os.Stat(configDir); os.IsNotExist(err) || !info.IsDir() {
			panic(fmt.Errorf("specified config dir '%s' is not a valid directory", configDir))
		}

		cfgs := config.LoadConfig(configDir, imposterConfig)
		store.PreloadStores(configDir, cfgs)

		configs = append(configs, cfgs...)
	}

	// Exit if no configuration files were found
	if len(configs) == 0 {
		logger.Errorf("no configuration files found in specified directories: %v", configDirs)
		os.Exit(1)
	} else {
		logger.Tracef("%d configs discovered", len(configs))
	}

	externalPlugins, err := external.StartExternalPlugins(imposterConfig, configs)
	if err != nil {
		panic(fmt.Errorf("error starting external plugins: %w", err))
	}

	plugins, err := plugin.LoadPlugins(configs, imposterConfig, externalPlugins)
	if err != nil {
		panic(fmt.Errorf("error loading plugins: %w", err))
	}

	// Pre-calculate resource IDs for all loaded configurations.
	// Note: we retrieve the config from each plugin to ensure it includes
	// any dynamic resources added by the plugin.
	for _, plg := range plugins {
		config.PreCalculateResourceID(plg.GetConfig())
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
