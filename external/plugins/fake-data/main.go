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
	impl := &FakeDataPlugin{
		logger: logger,
	}
	pluginMap := map[string]goplugin.Plugin{
		"fake-data": &shared.ExternalPlugin{Impl: impl},
	}

	logger.Trace("fake-data plugin initialising", "version", Version)

	handshakeConfig := goplugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "HANDLER_PLUGIN",
		MagicCookieValue: "imposter",
	}

	logger.Info("fake-data plugin started")
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
