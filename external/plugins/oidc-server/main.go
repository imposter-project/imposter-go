package main

import (
	"os"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/imposter-project/imposter-go/external/shared"
)

var Version = "dev"

var logger = hclog.New(&hclog.LoggerOptions{
	Level:      hclog.Trace,
	Output:     os.Stderr,
	JSONFormat: true,
})

func main() {
	impl := &OIDCServer{
		logger: logger,
	}
	pluginMap := map[string]goplugin.Plugin{
		"oidc-server": &shared.ExternalPlugin{Impl: impl},
	}

	logger.Trace("oidc-server plugin initialising", "version", Version)

	// handshakeConfigs are used to just do a basic handshake between
	// a plugin and host. If the handshake fails, a user-friendly error is shown.
	// This prevents users from executing bad plugins or executing a plugin
	// directory. It is a UX feature, not a security feature.
	handshakeConfig := goplugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "HANDLER_PLUGIN",
		MagicCookieValue: "imposter",
	}

	logger.Info("oidc-server plugin started")
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
