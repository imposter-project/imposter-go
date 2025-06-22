package swaggerui

import (
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/imposter-project/imposter-go/external/common"
	"net/rpc"
)

// SwaggerUIRPC is the RPC client
type SwaggerUIRPC struct{ client *rpc.Client }

func (s *SwaggerUIRPC) Handle(args common.HandlerArgs) string {
	var resp string
	err := s.client.Call("Plugin.Handle", args, &resp)
	if err != nil {
		// You usually want your interfaces to return errors. If they don't,
		// there isn't much other choice here.
		panic(err)
	}

	return resp
}

// Here is the RPC server that SwaggerUIRPC talks to, conforming to
// the requirements of net/rpc
type SwaggerUIRPCServer struct {
	// This is the real implementation
	Impl common.ExternalHandler
}

func (s *SwaggerUIRPCServer) Handle(args common.HandlerArgs, resp *string) error {
	*resp = s.Impl.Handle(args)
	return nil
}

// This is the implementation of plugin.Plugin so we can serve/consume this
//
// This has two methods: Server must return an RPC server for this plugin
// type. We construct a SwaggerUIRPCServer for this.
//
// Client must return an implementation of our interface that communicates
// over an RPC client. We return SwaggerUIRPC for this.
type SwaggerUIPlugin struct {
	// Impl Injection
	Impl common.ExternalHandler
}

func (p *SwaggerUIPlugin) Server(*goplugin.MuxBroker) (interface{}, error) {
	return &SwaggerUIRPCServer{Impl: p.Impl}, nil
}

func (SwaggerUIPlugin) Client(b *goplugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &SwaggerUIRPC{client: c}, nil
}
