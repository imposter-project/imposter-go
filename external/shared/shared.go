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

func (s *ExtPluginRPC) NormaliseRequest(args HandlerRequest) (NormaliseResponse, error) {
	var resp NormaliseResponse
	err := s.client.Call("Plugin.NormaliseRequest", args, &resp)
	if err != nil {
		return NormaliseResponse{}, fmt.Errorf("plugin.NormaliseRequest: %w", err)
	}
	return resp, nil
}

func (s *ExtPluginRPC) TransformResponse(args TransformRequest) (TransformResponseResult, error) {
	var resp TransformResponseResult
	err := s.client.Call("Plugin.TransformResponse", args, &resp)
	if err != nil {
		return TransformResponseResult{}, fmt.Errorf("plugin.TransformResponse: %w", err)
	}
	return resp, nil
}

func (s *ExtPluginRPC) GenerateFakeData(req FakeDataRequest) (FakeDataResponse, error) {
	var resp FakeDataResponse
	err := s.client.Call("Plugin.GenerateFakeData", req, &resp)
	if err != nil {
		return FakeDataResponse{}, fmt.Errorf("plugin.GenerateFakeData: %w", err)
	}
	return resp, nil
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

func (s *ExtPluginRPCServer) NormaliseRequest(args HandlerRequest, resp *NormaliseResponse) error {
	result, err := s.Impl.NormaliseRequest(args)
	if err != nil {
		return fmt.Errorf("plugin.NormaliseRequest: %w", err)
	}
	*resp = result
	return nil
}

func (s *ExtPluginRPCServer) TransformResponse(args TransformRequest, resp *TransformResponseResult) error {
	result, err := s.Impl.TransformResponse(args)
	if err != nil {
		return fmt.Errorf("plugin.TransformResponse: %w", err)
	}
	*resp = result
	return nil
}

func (s *ExtPluginRPCServer) GenerateFakeData(req FakeDataRequest, resp *FakeDataResponse) error {
	result, err := s.Impl.GenerateFakeData(req)
	if err != nil {
		return fmt.Errorf("plugin.GenerateFakeData: %w", err)
	}
	*resp = result
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
