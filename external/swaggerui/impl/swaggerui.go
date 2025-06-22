package main

import (
	"github.com/imposter-project/imposter-go/external/common"
	"github.com/imposter-project/imposter-go/external/swaggerui"
	"os"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
)

type SwaggerUI struct {
	pluginName string
	logger     hclog.Logger
}

func (s *SwaggerUI) Handle(args common.HandlerArgs) string {
	s.logger.Debug(s.pluginName+" handling swagger ui", "method", args.Method, "path", args.Path)
	return "Swagger UI response for " + args.Method + " " + args.Path
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "hello",
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	impl := &SwaggerUI{
		pluginName: "swaggerui",
		logger:     logger,
	}
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]goplugin.Plugin{
		"swaggerui": &swaggerui.SwaggerUIPlugin{Impl: impl},
	}

	logger.Debug("swaggerui plugin initialising")

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
