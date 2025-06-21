package swaggerui

import (
	goplugin "github.com/hashicorp/go-plugin"
	"github.com/imposter-project/imposter-go/external/common"
	"net/rpc"
)

type RPCArgs struct {
	Path string
}

// Here is an implementation that talks over RPC
type SwaggerUIRPC struct{ client *rpc.Client }

func (s *SwaggerUIRPC) Handle(path string) string {
	var resp string
	err := s.client.Call("Plugin.Handle", RPCArgs{Path: path}, &resp)
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

func (s *SwaggerUIRPCServer) Handle(args RPCArgs, resp *string) error {
	path := args.Path
	*resp = s.Impl.Handle(path)
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
