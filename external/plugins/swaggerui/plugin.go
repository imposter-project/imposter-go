package main

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/imposter-project/imposter-go/external/shared"
	"os"
	"strings"
)

var Version = "dev"

type SwaggerUI struct {
	logger hclog.Logger
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	impl := &SwaggerUI{
		logger: logger,
	}
	pluginMap := map[string]goplugin.Plugin{
		"swaggerui": &shared.ExternalPlugin{Impl: impl},
	}

	logger.Debug("swaggerui plugin initialising", "version", Version, "path", specPrefixPath)

	// handshakeConfigs are used to just do a basic handshake between
	// a plugin and host. If the handshake fails, a user-friendly error is shown.
	// This prevents users from executing bad plugins or executing a plugin
	// directory. It is a UX feature, not a security feature.
	handshakeConfig := goplugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "HANDLER_PLUGIN",
		MagicCookieValue: "imposter",
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}

func (s *SwaggerUI) Configure(configs []shared.LightweightConfig) error {
	s.logger.Trace("generating spec config")
	if err := generateSpecConfig(configs); err != nil {
		return fmt.Errorf("could not generate swagger UI plugin config: %w", err)
	}

	s.logger.Trace("generating index page")
	if err := generateIndexPage(); err != nil {
		return fmt.Errorf("could not generate index page: %w", err)
	}
	return nil
}

func (s *SwaggerUI) Handle(args shared.HandlerRequest) shared.HandlerResponse {
	path := args.Path
	if !strings.EqualFold(args.Method, "get") {
		return shared.HandlerResponse{StatusCode: 405, Body: []byte("Method Not Allowed")}
	}
	if !strings.HasPrefix(path, specPrefixPath) {
		return shared.HandlerResponse{StatusCode: 404, Body: []byte("File Not Found")}
	}

	if response := serveRawSpec(path); response != nil {
		return *response
	} else {
		return serveStaticContent(path)
	}
}
