package shared

import (
	"fmt"
	goplugin "github.com/hashicorp/go-plugin"
	"net/rpc"
)

// ExtPluginRPC is the RPC client
type ExtPluginRPC struct{ client *rpc.Client }

func (s *ExtPluginRPC) Configure(configs []LightweightConfig) error {
	var resp struct{} // No response needed
	err := s.client.Call("Plugin.Configure", configs, &resp)
	if err != nil {
		return fmt.Errorf("plugin.Configure: %w", err)
	}
	return nil
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

// ExtPluginRPCServer is the RPC server that ExtPluginRPC talks to, conforming to
// the requirements of net/rpc
type ExtPluginRPCServer struct {
	// This is the real implementation
	Impl ExternalHandler
}

func (s *ExtPluginRPCServer) Configure(configs []LightweightConfig, resp *struct{}) error {
	err := s.Impl.Configure(configs)
	if err != nil {
		return fmt.Errorf("plugin.Configure: %w", err)
	}
	*resp = struct{}{} // No response needed
	return nil
}

func (s *ExtPluginRPCServer) Handle(args HandlerRequest, resp *HandlerResponse) error {
	*resp = s.Impl.Handle(args)
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
