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

// InitialiseImposter performs common initialisation tasks for all adapters.
// It returns an error rather than panicking for user-facing problems (such as
// missing or invalid configuration) so callers can present a concise message
// and exit with a non-zero status instead of dumping a stack trace.
func InitialiseImposter(configDirArg string) (*config.ImposterConfig, []plugin.Plugin, error) {
	logger.Infof("starting imposter-go %s...", version.Version)

	imposterConfig := config.LoadImposterConfig()
	configDirs, err := getConfigDirs(configDirArg)
	if err != nil {
		return nil, nil, err
	}

	store.InitStoreProvider()

	var configs []config.Config
	for _, configDir := range configDirs {
		if info, err := os.Stat(configDir); err != nil || !info.IsDir() {
			return nil, nil, fmt.Errorf("specified config dir '%s' is not a valid directory", configDir)
		}

		cfgs, err := config.LoadConfig(configDir, imposterConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load configuration from '%s': %w", configDir, err)
		}
		if err := store.PreloadStores(configDir, cfgs); err != nil {
			return nil, nil, err
		}

		configs = append(configs, cfgs...)
	}

	// Fail if no configuration files were found
	if len(configs) == 0 {
		return nil, nil, fmt.Errorf("no configuration files found in specified directories: %v", configDirs)
	}
	logger.Tracef("%d configs discovered", len(configs))

	externalPlugins, err := external.StartExternalPlugins(imposterConfig, configs)
	if err != nil {
		return nil, nil, fmt.Errorf("error starting external plugins: %w", err)
	}

	plugins, err := plugin.LoadPlugins(configs, imposterConfig, externalPlugins)
	if err != nil {
		return nil, nil, fmt.Errorf("error loading plugins: %w", err)
	}

	// Pre-calculate resource IDs for all loaded configurations.
	// Note: we retrieve the config from each plugin to ensure it includes
	// any dynamic resources added by the plugin.
	for _, plg := range plugins {
		config.PreCalculateResourceID(plg.GetConfig())
	}

	return imposterConfig, plugins, nil
}

func getConfigDirs(configDirArg string) ([]string, error) {
	var configDirRaw string
	if configDirArg != "" {
		configDirRaw = configDirArg
	} else {
		configDirRaw = os.Getenv("IMPOSTER_CONFIG_DIR")
		if configDirRaw == "" {
			return nil, fmt.Errorf("config directory path must be provided either as an argument or via the IMPOSTER_CONFIG_DIR environment variable")
		}
	}
	configDirs := strings.Split(configDirRaw, ",")
	return configDirs, nil
}
