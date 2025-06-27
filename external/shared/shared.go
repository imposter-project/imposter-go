package shared

import (
	"fmt"
	goplugin "github.com/hashicorp/go-plugin"
	"net/rpc"
)

// SwaggerUIRPC is the RPC client
type SwaggerUIRPC struct{ client *rpc.Client }

func (s *SwaggerUIRPC) Configure(configs []LightweightConfig) error {
	var resp struct{} // No response needed
	err := s.client.Call("Plugin.Configure", configs, &resp)
	if err != nil {
		return fmt.Errorf("plugin.Configure: %w", err)
	}
	return nil
}

func (s *SwaggerUIRPC) Handle(args HandlerRequest) HandlerResponse {
	var resp HandlerResponse
	err := s.client.Call("Plugin.Handle", args, &resp)
	if err != nil {
		// TODO return an error instead of panic
		panic(err)
	}

	return resp
}

// SwaggerUIRPCServer is the RPC server that SwaggerUIRPC talks to, conforming to
// the requirements of net/rpc
type SwaggerUIRPCServer struct {
	// This is the real implementation
	Impl ExternalHandler
}

func (s *SwaggerUIRPCServer) Configure(configs []LightweightConfig, resp *struct{}) error {
	err := s.Impl.Configure(configs)
	if err != nil {
		return fmt.Errorf("plugin.Configure: %w", err)
	}
	*resp = struct{}{} // No response needed
	return nil
}

func (s *SwaggerUIRPCServer) Handle(args HandlerRequest, resp *HandlerResponse) error {
	*resp = s.Impl.Handle(args)
	return nil
}

// SwaggerUIPlugin is the implementation of plugin.Plugin
//
// This must have two methods:
//
// 1. Server must return an RPC server for this plugin
// type. We construct a SwaggerUIRPCServer for this.
//
// 2. Client must return an implementation of our interface that communicates
// over an RPC client. We return SwaggerUIRPC for this.
type SwaggerUIPlugin struct {
	// Impl Injection
	Impl ExternalHandler
}

func (p *SwaggerUIPlugin) Server(*goplugin.MuxBroker) (interface{}, error) {
	return &SwaggerUIRPCServer{Impl: p.Impl}, nil
}

func (SwaggerUIPlugin) Client(b *goplugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &SwaggerUIRPC{client: c}, nil
}
