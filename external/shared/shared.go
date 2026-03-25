package shared

import (
	"fmt"
	goplugin "github.com/hashicorp/go-plugin"
	"net/rpc"
)

// ExtPluginRPC is the RPC client
type ExtPluginRPC struct{ client *rpc.Client }

func (s *ExtPluginRPC) Configure(cfg ExternalConfig) (PluginCapabilities, error) {
	var resp PluginCapabilities
	err := s.client.Call("Plugin.Configure", cfg, &resp)
	if err != nil {
		return PluginCapabilities{}, fmt.Errorf("plugin.Configure: %w", err)
	}
	return resp, nil
}

func (s *ExtPluginRPC) Handle(args HandlerRequest) HandlerResponse {
	var resp HandlerResponse
	err := s.client.Call("Plugin.Handle", args, &resp)
	if err != nil {
		// TODO return an error instead of panic
		panic(err)
	}
	return resp
}

func (s *ExtPluginRPC) GenerateFakeData(req FakeDataRequest) FakeDataResponse {
	var resp FakeDataResponse
	err := s.client.Call("Plugin.GenerateFakeData", req, &resp)
	if err != nil {
		return FakeDataResponse{}
	}
	return resp
}

// ExtPluginRPCServer is the RPC server that ExtPluginRPC talks to, conforming to
// the requirements of net/rpc
type ExtPluginRPCServer struct {
	// This is the real implementation
	Impl ExternalHandler
}

func (s *ExtPluginRPCServer) Configure(cfg ExternalConfig, resp *PluginCapabilities) error {
	caps, err := s.Impl.Configure(cfg)
	if err != nil {
		return fmt.Errorf("plugin.Configure: %w", err)
	}
	*resp = caps
	return nil
}

func (s *ExtPluginRPCServer) Handle(args HandlerRequest, resp *HandlerResponse) error {
	*resp = s.Impl.Handle(args)
	return nil
}

func (s *ExtPluginRPCServer) GenerateFakeData(req FakeDataRequest, resp *FakeDataResponse) error {
	*resp = s.Impl.GenerateFakeData(req)
	return nil
}

// ExternalPlugin is the implementation of plugin.Plugin
//
// This must have two methods:
//
// 1. Server must return an RPC server for this plugin
// type. We construct a ExtPluginRPCServer for this.
//
// 2. Client must return an implementation of our interface that communicates
// over an RPC client. We return ExtPluginRPC for this.
type ExternalPlugin struct {
	// FilePath is the path to the plugin file.
	FilePath string

	// Impl Injection
	Impl ExternalHandler
}

func (p *ExternalPlugin) Server(*goplugin.MuxBroker) (interface{}, error) {
	return &ExtPluginRPCServer{Impl: p.Impl}, nil
}

func (ExternalPlugin) Client(b *goplugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ExtPluginRPC{client: c}, nil
}
