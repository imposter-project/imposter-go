package main

import (
	"embed"
	"errors"
	"fmt"
	"github.com/imposter-project/imposter-go/external/handler"
	"github.com/imposter-project/imposter-go/external/plugins/swaggerui"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
)

//go:embed www/*
var www embed.FS

type SwaggerUI struct {
	logger hclog.Logger
}

var specPrefixPath string

func init() {
	specPrefixPath = os.Getenv("IMPOSTER_OPENAPI_SPEC_PATH_PREFIX")
	if specPrefixPath == "" {
		specPrefixPath = "/_spec"
	}
}

func (s *SwaggerUI) Handle(args handler.HandlerRequest) handler.HandlerResponse {
	path := args.Path
	s.logger.Debug("handling swagger ui request", "method", args.Method, "path", path)
	if !strings.EqualFold(args.Method, "get") {
		return handler.HandlerResponse{StatusCode: 405, Body: []byte("Method Not Allowed")}
	}
	if !strings.HasPrefix(path, specPrefixPath) {
		return handler.HandlerResponse{StatusCode: 404, Body: []byte("File Not Found")}
	} else {
		path = strings.TrimPrefix(path, specPrefixPath)
	}

	respHeaders := make(map[string]string)
	if path == "/" {
		path = "/index.html"
		respHeaders["Content-Type"] = "text/html; charset=utf-8"
	}
	file, err := www.ReadFile("www" + path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return handler.HandlerResponse{StatusCode: 404, Body: []byte("File Not Found")}
		}
		return handler.HandlerResponse{
			StatusCode: 500,
			Body:       []byte(fmt.Sprintf("Error reading file: %s - %v", path, err.Error())),
		}
	}
	return handler.HandlerResponse{
		StatusCode: 200,
		Headers:    respHeaders,
		Body:       file,
	}
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "HANDLER_PLUGIN",
	MagicCookieValue: "imposter",
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
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]goplugin.Plugin{
		"swaggerui": &swaggerui.SwaggerUIPlugin{Impl: impl},
	}

	logger.Debug("swaggerui plugin initialising", "path", specPrefixPath)

	if logger.IsTrace() {
		entries, err := www.ReadDir("www")
		if err != nil {
			panic(fmt.Errorf("failed to read static files: %v", err))
		}
		for _, entry := range entries {
			logger.Trace(entry.Name())
		}
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
