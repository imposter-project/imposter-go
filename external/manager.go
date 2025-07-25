package external

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/imposter-project/imposter-go/external/shared"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/version"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/imposter-project/imposter-go/plugin"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var pluginMap map[string]goplugin.Plugin

type LoadedPlugin struct {
	name   string
	client *goplugin.Client
	impl   *shared.ExternalHandler
}

var pluginDir string
var hasPlugins bool
var loaded []LoadedPlugin

// StartExternalPlugins initialises and starts all external plugins defined in the pluginMap,
// passing the current configuration to each plugin.
func StartExternalPlugins(imposterConfig *config.ImposterConfig, plugins []plugin.Plugin) error {
	if err := discoverPlugins(); err != nil {
		return fmt.Errorf("failed to discover plugins: %v", err)
	}
	if !hasPlugins {
		logger.Tracef("no external plugins found to load")
		return nil
	}
	logger.Tracef("external plugins enabled")

	hclogger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: os.Stdout,
		Level:  getHcLogLevel(),
	})

	cfg := buildConfig(imposterConfig, plugins)

	for pluginName, p := range pluginMap {
		plg := p.(*shared.ExternalPlugin)
		err := start(cfg, pluginName, plg, hclogger)
		if err != nil {
			return fmt.Errorf("failed to start plugin %s: %v", pluginName, err)
		}
	}

	logger.Debugf("successfully loaded %d external plugins", len(loaded))
	return nil
}

func buildConfig(imposterConfig *config.ImposterConfig, plugins []plugin.Plugin) shared.ExternalConfig {
	var lightweight []shared.LightweightConfig
	for _, plg := range plugins {
		cfg := plg.GetConfig()
		lightweight = append(lightweight, shared.LightweightConfig{
			ConfigDir: plg.GetConfigDir(),
			Plugin:    cfg.Plugin,
			SpecFile:  cfg.SpecFile,
		})
	}

	cfg := shared.ExternalConfig{
		Server: shared.ServerConfig{
			URL: imposterConfig.ServerUrl,
		},
		Configs: lightweight,
	}
	return cfg
}

func getHcLogLevel() hclog.Level {
	switch logger.GetCurrentLevel() {
	case logger.TRACE:
		return hclog.Trace
	case logger.DEBUG:
		return hclog.Debug
	case logger.INFO:
		return hclog.Info
	case logger.WARN:
		return hclog.Warn
	case logger.ERROR:
		return hclog.Error
	default:
		return hclog.NoLevel
	}
}

// start starts and configures an external plugin
func start(cfg shared.ExternalConfig, pluginName string, plg *shared.ExternalPlugin, hclogger hclog.Logger) error {
	logger.Debugf("loading external plugin: %s", pluginName)

	singlePlugin := map[string]goplugin.Plugin{
		pluginName: plg,
	}

	// launch the plugin process
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         singlePlugin,
		Cmd:             exec.Command(plg.FilePath),
		Logger:          hclogger,
	})

	// connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		return fmt.Errorf("error connecting to plugin %s: %v", pluginName, err)
	}

	// request the plugin
	raw, err := rpcClient.Dispense(pluginName)
	if err != nil {
		return fmt.Errorf("error dispensing plugin %s: %v", pluginName, err)
	}
	impl := raw.(shared.ExternalHandler)

	err = impl.Configure(cfg)
	if err != nil {
		return fmt.Errorf("failed to configure plugin %s: %v", pluginName, err)
	}

	loaded = append(loaded, LoadedPlugin{
		name:   pluginName,
		client: client,
		impl:   &impl,
	})
	return nil
}

// InvokeExternalHandlers calls the external plugins with the provided handler request
// and returns the first successful response, or none if no plugin handled the request.
func InvokeExternalHandlers(args shared.HandlerRequest) *shared.HandlerResponse {
	if !hasPlugins {
		return nil
	}

	var resp shared.HandlerResponse
	for _, l := range loaded {
		impl := *l.impl
		resp = impl.Handle(args)
		if resp.StatusCode >= 100 && resp.StatusCode < 300 {
			logger.Debugf("response from plugin %s: status=%d body=%d bytes", l.name, resp.StatusCode, len(resp.Body))
			break
		} else if resp.StatusCode == 0 || resp.StatusCode == 404 {
			// plugin did not handle the request, continue to the next plugin
			logger.Tracef("plugin %s did not handle the request, continuing to next plugin", l.name)
			continue
		} else {
			logger.Errorf("error response from plugin %s: status=%d body=%d bytes", l.name, resp.StatusCode, len(resp.Body))
			break
		}
	}
	return &resp
}

// StopExternalPlugins stops all loaded external plugins by killing their processes.
func StopExternalPlugins() {
	if !hasPlugins {
		return
	}
	for _, l := range loaded {
		logger.Debugf("unloading external plugin: %s", l.name)
		l.client.Kill()
	}
	loaded = nil
}

// discoverPlugins finds the directory from which plugins are loaded.
func discoverPlugins() error {
	if os.Getenv("IMPOSTER_EXTERNAL_PLUGINS") != "true" {
		logger.Tracef("external plugins are disabled by environment variable IMPOSTER_EXTERNAL_PLUGINS")
		hasPlugins = false
		return nil
	}

	var envPluginDir = os.Getenv("IMPOSTER_PLUGIN_DIR")
	if len(envPluginDir) > 0 {
		pluginDir = path.Clean(envPluginDir)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
		pluginDir = filepath.Join(homeDir, ".imposter", "plugins")
	}

	discovered, err := listPluginsInDir(pluginDir, true)
	if err != nil {
		return err
	}
	pluginMap = discovered
	hasPlugins = len(pluginMap) > 0
	return nil
}

func listPluginsInDir(dir string, checkVersionedSubDir bool) (map[string]goplugin.Plugin, error) {
	logger.Tracef("listing plugins in dir: %s", dir)
	discovered := make(map[string]goplugin.Plugin)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins in directory %s: %v", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if !checkVersionedSubDir {
				continue
			}
			if entry.Name() != version.Version {
				logger.Tracef("skipping subdirectory %s, not matching version %s", entry.Name(), version.Version)
				continue
			}
			subdir := filepath.Join(dir, entry.Name())
			subdirPlugins, err := listPluginsInDir(subdir, false)
			if err != nil {
				return nil, err
			}
			for name, p := range subdirPlugins {
				discovered[name] = p
			}
		}

		if !strings.HasPrefix(entry.Name(), "plugin-") {
			continue
		}

		pluginName := getPluginNameFromFileName(entry.Name())
		logger.Debugf("found plugin: %s", pluginName)
		discovered[pluginName] = &shared.ExternalPlugin{FilePath: filepath.Join(dir, entry.Name())}
	}
	return discovered, nil
}

func getPluginNameFromFileName(fileName string) string {
	return strings.TrimSuffix(strings.TrimPrefix(fileName, "plugin-"), ".exe")
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user-friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "HANDLER_PLUGIN",
	MagicCookieValue: "imposter",
}
