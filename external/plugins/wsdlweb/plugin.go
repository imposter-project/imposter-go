package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/imposter-project/imposter-go/external/shared"
)

var Version = "dev"

type WSDLWeb struct {
	logger hclog.Logger
}

var logger = hclog.New(&hclog.LoggerOptions{
	Level:      hclog.Trace,
	Output:     os.Stderr,
	JSONFormat: true,
})

var config shared.ExternalConfig

func main() {
	impl := &WSDLWeb{
		logger: logger,
	}
	pluginMap := map[string]goplugin.Plugin{
		"wsdlweb": &shared.ExternalPlugin{Impl: impl},
	}

	logger.Trace("wsdlweb plugin initialising", "version", Version, "prefixPath", wsdlPrefixPath)

	handshakeConfig := goplugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "HANDLER_PLUGIN",
		MagicCookieValue: "imposter",
	}

	logger.Info("wsdlweb WSDL viewer hosted at", "path", wsdlPrefixPath+"/")
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}

func (w *WSDLWeb) Configure(cfg shared.ExternalConfig) (shared.PluginCapabilities, error) {
	config = cfg

	w.logger.Trace("generating WSDL config")
	if err := generateWSDLConfig(config.Configs); err != nil {
		return shared.PluginCapabilities{}, fmt.Errorf("could not generate WSDL Web plugin config: %w", err)
	}

	w.logger.Trace("generating index page")
	if err := generateInitialiser(); err != nil {
		return shared.PluginCapabilities{}, fmt.Errorf("could not generate initialiser: %w", err)
	}
	return shared.PluginCapabilities{HandleRequests: true}, nil
}

func (w *WSDLWeb) GenerateFakeData(req shared.FakeDataRequest) shared.FakeDataResponse {
	return shared.FakeDataResponse{}
}

func (w *WSDLWeb) Handle(args shared.HandlerRequest) shared.HandlerResponse {
	path := args.Path
	if !strings.HasPrefix(path, wsdlPrefixPath) {
		// not handled
		return shared.HandlerResponse{StatusCode: 0}
	}

	if !strings.EqualFold(args.Method, "get") {
		return shared.HandlerResponse{StatusCode: 405, Body: []byte("Method Not Allowed")}
	}
	if response := serveRawWSDL(path); response != nil {
		return *response
	} else {
		return serveStaticContent(path)
	}
}
