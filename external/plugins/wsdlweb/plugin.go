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

func (w *WSDLWeb) GenerateFakeData(_ shared.FakeDataRequest) (shared.FakeDataResponse, error) {
	return shared.FakeDataResponse{}, nil
}

func (w *WSDLWeb) NormaliseRequest(args shared.HandlerRequest) (shared.NormaliseResponse, error) {
	if !strings.HasPrefix(args.Path, wsdlPrefixPath) {
		return shared.NormaliseResponse{Skip: true}, nil
	}
	return shared.NormaliseResponse{}, nil
}

func (w *WSDLWeb) TransformResponse(args shared.TransformRequest) (shared.TransformResponseResult, error) {
	if args.Handled {
		return shared.TransformResponseResult{
			StatusCode: args.StatusCode,
			Headers:    args.ResponseHeaders,
			Body:       args.ResponseBody,
		}, nil
	}

	// Pipeline did not match — serve WSDL content
	if !strings.EqualFold(args.Method, "get") {
		return shared.TransformResponseResult{StatusCode: 405, Body: []byte("Method Not Allowed")}, nil
	}
	if response := serveRawWSDL(args.Path); response != nil {
		return shared.TransformResponseResult{
			StatusCode: response.StatusCode,
			Headers:    response.Headers,
			Body:       response.Body,
		}, nil
	}
	resp := serveStaticContent(args.Path)
	return shared.TransformResponseResult{
		StatusCode: resp.StatusCode,
		Headers:    resp.Headers,
		Body:       resp.Body,
	}, nil
}
