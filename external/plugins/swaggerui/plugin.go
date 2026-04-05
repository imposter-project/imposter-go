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

var logger = hclog.New(&hclog.LoggerOptions{
	Level:      hclog.Trace,
	Output:     os.Stderr,
	JSONFormat: true,
})

var config shared.ExternalConfig

func main() {
	impl := &SwaggerUI{
		logger: logger,
	}
	pluginMap := map[string]goplugin.Plugin{
		"swaggerui": &shared.ExternalPlugin{Impl: impl},
	}

	logger.Trace("swaggerui plugin initialising", "version", Version, "prefixPath", specPrefixPath)

	// handshakeConfigs are used to just do a basic handshake between
	// a plugin and host. If the handshake fails, a user-friendly error is shown.
	// This prevents users from executing bad plugins or executing a plugin
	// directory. It is a UX feature, not a security feature.
	handshakeConfig := goplugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "HANDLER_PLUGIN",
		MagicCookieValue: "imposter",
	}

	logger.Info("swaggerui spec hosted at", "path", specPrefixPath+"/")
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}

func (s *SwaggerUI) Configure(cfg shared.ExternalConfig) (shared.PluginCapabilities, error) {
	config = cfg

	s.logger.Trace("generating spec config")
	if err := generateSpecConfig(config.Configs); err != nil {
		return shared.PluginCapabilities{}, fmt.Errorf("could not generate swagger UI plugin config: %w", err)
	}

	s.logger.Trace("generating index page")
	if err := generateInitialiser(); err != nil {
		return shared.PluginCapabilities{}, fmt.Errorf("could not generate initialiser: %w", err)
	}
	return shared.PluginCapabilities{HandleRequests: true}, nil
}

func (s *SwaggerUI) GenerateFakeData(_ shared.FakeDataRequest) (shared.FakeDataResponse, error) {
	return shared.FakeDataResponse{}, nil
}

func (s *SwaggerUI) NormaliseRequest(args shared.HandlerRequest) (shared.NormaliseResponse, error) {
	if !strings.HasPrefix(args.Path, specPrefixPath) {
		return shared.NormaliseResponse{Skip: true}, nil
	}
	return shared.NormaliseResponse{}, nil
}

func (s *SwaggerUI) TransformResponse(args shared.TransformRequest) (shared.TransformResponseResult, error) {
	if args.Handled {
		// Pipeline matched a resource — pass through its response
		return shared.TransformResponseResult{
			StatusCode: args.StatusCode,
			Headers:    args.ResponseHeaders,
			Body:       args.ResponseBody,
		}, nil
	}

	// Pipeline did not match — serve SwaggerUI content
	if !strings.EqualFold(args.Method, "get") {
		return shared.TransformResponseResult{StatusCode: 405, Body: []byte("Method Not Allowed")}, nil
	}
	if response := serveRawSpec(config.Server, args.Path); response != nil {
		return shared.TransformResponseResult{
			StatusCode: response.StatusCode,
			Headers:    response.Headers,
			Body:       response.Body,
			FileName:   response.FileName,
		}, nil
	}
	resp := serveStaticContent(args.Path)
	return shared.TransformResponseResult{
		StatusCode: resp.StatusCode,
		Headers:    resp.Headers,
		Body:       resp.Body,
		FileName:   resp.FileName,
	}, nil
}
