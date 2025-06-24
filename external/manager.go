package external

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/imposter-project/imposter-go/external/handler"
	"github.com/imposter-project/imposter-go/external/plugins/swaggerui"
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
	impl   handler.ExternalHandler
}

var hasPlugins bool
var loaded []LoadedPlugin

func StartExternalPlugins() {
	hasPlugins = len(os.Getenv("IMPOSTER_PLUGIN_DIR")) > 0 && len(pluginMap) > 0
	if !hasPlugins {
		logger.Tracef("no external plugins found to load")
		return
	}

	// Create an hclog.Logger
	hclogger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	for pluginName := range pluginMap {
		start(pluginName, hclogger)
	}
}

func start(pluginName string, hclogger hclog.Logger) {
	logger.Debugf("loading external plugin: %s", pluginName)
	pluginPath := path.Join(getPluginDir(), "plugin-"+pluginName)

	// We're a host! Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		Cmd:             exec.Command(pluginPath),
		Logger:          hclogger,
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		log.Fatal(err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(pluginName)
	if err != nil {
		log.Fatal(err)
	}

	// We should have a plugin stub now! This feels like a normal interface
	// implementation but is in fact over an RPC connection.
	impl := raw.(handler.ExternalHandler)

	loaded = append(loaded, LoadedPlugin{
		name:   pluginName,
		client: client,
		impl:   impl,
	})
}

func InvokeExternalHandlers(args handler.HandlerRequest) *handler.HandlerResponse {
	if !hasPlugins {
		return nil
	}

	var resp handler.HandlerResponse
	for _, l := range loaded {
		resp = l.impl.Handle(args)
		if resp.StatusCode >= 100 && resp.StatusCode < 300 {
			logger.Debugf("response from plugin %s: status=%d body=%d bytes", l.name, resp.StatusCode, len(resp.Body))
			break
		} else if resp.StatusCode == 404 {
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

func StopExternalPlugins() {
	if !hasPlugins {
		return
	}
	for _, l := range loaded {
		logger.Debugf("unloading external plugin: %s", l.name)
		l.client.Kill()
	}
}

func getPluginDir() string {
	envPath := os.Getenv("IMPOSTER_PLUGIN_DIR")
	if envPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("failed to get user home directory: %v", err)
		}
		envPath = path.Join(homeDir, ".imposter", "plugins")
	}
	return envPath
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "HANDLER_PLUGIN",
	MagicCookieValue: "imposter",
}
