package external

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/imposter-project/imposter-go/external/handler"
	"github.com/imposter-project/imposter-go/external/plugins/swaggerui"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"log"
	"os"
	"os/exec"
	"path"
)

// pluginMap is the map of plugins we can dispense.
var pluginMap = map[string]plugin.Plugin{
	"swaggerui": &swaggerui.SwaggerUIPlugin{},
}

type LoadedPlugin struct {
	name   string
	client *plugin.Client
	impl   *handler.ExternalHandler
}

var pluginDir string
var hasPlugins bool
var loaded []LoadedPlugin

// StartExternalPlugins initialises and starts all external plugins defined in the pluginMap,
// passing the current configuration to each plugin.
func StartExternalPlugins(configs []config.Config) error {
	discoverPluginDir()
	hasPlugins = len(pluginDir) > 0 && len(pluginMap) > 0
	if !hasPlugins {
		logger.Tracef("no external plugins found to load")
		return nil
	}
	logger.Tracef("external plugins enabled")

	hclogger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	var lightweight []handler.LightweightConfig
	for _, cfg := range configs {
		lightweight = append(lightweight, handler.LightweightConfig{
			Plugin:   cfg.Plugin,
			SpecFile: cfg.SpecFile,
		})
	}

	for pluginName := range pluginMap {
		err := start(pluginName, hclogger, lightweight)
		if err != nil {
			return fmt.Errorf("failed to start plugin %s: %v", pluginName, err)
		}
	}

	logger.Debugf("successfully loaded %d external plugins", len(loaded))
	return nil
}

// start initialises and starts a single external plugin by its name.
func start(pluginName string, hclogger hclog.Logger, configs []handler.LightweightConfig) error {
	logger.Debugf("loading external plugin: %s", pluginName)
	pluginPath := path.Join(pluginDir, "plugin-"+pluginName)

	// launch the plugin process
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		Cmd:             exec.Command(pluginPath),
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
	impl := raw.(handler.ExternalHandler)

	err = impl.Configure(configs)
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
func InvokeExternalHandlers(args handler.HandlerRequest) *handler.HandlerResponse {
	if !hasPlugins {
		return nil
	}

	var resp handler.HandlerResponse
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
}

// discoverPluginDir finds the directory from which plugins are loaded.
func discoverPluginDir() {
	var envPluginDir = os.Getenv("IMPOSTER_PLUGIN_DIR")
	if len(envPluginDir) > 0 {
		pluginDir = path.Clean(envPluginDir)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get user home directory: %v", err)
		}
		pluginDir = path.Join(homeDir, ".imposter", "plugins")
	}
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user-friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "HANDLER_PLUGIN",
	MagicCookieValue: "imposter",
}
