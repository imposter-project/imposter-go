package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// GRPCPlugin implements the ExternalHandler interface for mocking gRPC services.
type GRPCPlugin struct {
	logger  hclog.Logger
	methods map[string]*methodDescriptors // gRPC path -> message descriptors
}

// methodRef is serialised into NormaliseResponse.Metadata so TransformResponse
// knows which proto method descriptor to use for encoding.
type methodRef struct {
	GRPCPath string `json:"grpcPath"`
}

func (g *GRPCPlugin) Configure(cfg shared.ExternalConfig) (shared.PluginCapabilities, error) {
	g.logger.Trace("configuring gRPC plugin")

	for _, lwConfig := range cfg.Configs {
		if len(lwConfig.PluginConfig) == 0 {
			continue
		}
		if g.methods != nil {
			g.logger.Warn("multiple gRPC config blocks found, using the first one")
			break
		}

		g.logger.Debug("loading gRPC config from plugin config block")
		config, err := loadGRPCConfig(lwConfig.PluginConfig)
		if err != nil {
			return shared.PluginCapabilities{}, fmt.Errorf("failed to load gRPC config: %w", err)
		}

		methods, err := parseProtoFiles(lwConfig.ConfigDir, config.ProtoFiles, g.logger)
		if err != nil {
			return shared.PluginCapabilities{}, fmt.Errorf("failed to parse proto files: %w", err)
		}
		g.methods = methods

		g.logger.Info("gRPC plugin configured",
			"protoFiles", len(config.ProtoFiles),
			"methods", len(methods),
		)
	}

	if g.methods == nil {
		return shared.PluginCapabilities{}, fmt.Errorf("no gRPC configuration provided")
	}

	return shared.PluginCapabilities{HandleRequests: true}, nil
}

func (g *GRPCPlugin) NormaliseRequest(args shared.HandlerRequest) (shared.NormaliseResponse, error) {
	// Only handle gRPC requests (Content-Type: application/grpc, method: POST)
	contentType := args.Headers["Content-Type"]
	if contentType == "" {
		contentType = args.Headers["content-type"]
	}
	if !strings.HasPrefix(contentType, "application/grpc") {
		return shared.NormaliseResponse{Skip: true}, nil
	}
	if !strings.EqualFold(args.Method, "POST") {
		return shared.NormaliseResponse{Skip: true}, nil
	}

	g.logger.Debug("normalising gRPC request", "path", args.Path)

	// Look up the method descriptor for this gRPC path
	md, ok := g.methods[args.Path]
	if !ok {
		g.logger.Debug("no proto definition for gRPC method", "path", args.Path)
		// Still handle it — TransformResponse will return UNIMPLEMENTED
		metadata, _ := json.Marshal(methodRef{GRPCPath: args.Path})
		return shared.NormaliseResponse{Metadata: metadata}, nil
	}

	// Store the gRPC path in metadata for TransformResponse
	metadata, _ := json.Marshal(methodRef{GRPCPath: args.Path})

	// Decode gRPC frame → protobuf → JSON so the core pipeline can
	// use body matching and templating on the request
	var jsonBody []byte
	if len(args.Body) >= grpcFrameHeaderSize {
		protoBytes, err := DecodeGRPCFrame(args.Body)
		if err == nil && len(protoBytes) > 0 {
			msg := dynamicpb.NewMessage(md.input)
			if err := proto.Unmarshal(protoBytes, msg); err == nil {
				jsonBody, _ = protojson.Marshal(msg)
			}
		}
	}

	return shared.NormaliseResponse{
		Body:     jsonBody,
		Metadata: metadata,
	}, nil
}

func (g *GRPCPlugin) TransformResponse(args shared.TransformRequest) (shared.TransformResponseResult, error) {
	// Decode metadata to find the gRPC path
	var ref methodRef
	if len(args.Metadata) > 0 {
		json.Unmarshal(args.Metadata, &ref)
	}

	if !args.Handled {
		// Pipeline did not match — return gRPC UNIMPLEMENTED
		g.logger.Debug("no resource matched for gRPC method", "path", ref.GRPCPath)
		return shared.TransformResponseResult{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/grpc",
			},
			Trailers: map[string]string{
				"Grpc-Status":  strconv.Itoa(12), // UNIMPLEMENTED
				"Grpc-Message": "no resource configured for: " + ref.GRPCPath,
			},
			Body: EncodeGRPCFrame(nil),
		}, nil
	}

	// Use the pipeline's status code as the gRPC status when it falls
	// within the valid gRPC range (1–16). Values like 200 are HTTP
	// defaults from the pipeline and map to gRPC OK (0).
	grpcStatus := "0"
	if args.StatusCode >= 1 && args.StatusCode <= 16 {
		grpcStatus = strconv.Itoa(args.StatusCode)
	}

	// If no response body, return an empty gRPC frame with the status.
	// This is typical for error responses (e.g. NOT_FOUND) that carry
	// no message body.
	if len(args.ResponseBody) == 0 {
		return shared.TransformResponseResult{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/grpc",
			},
			Trailers: map[string]string{
				"Grpc-Status": grpcStatus,
			},
			Body: EncodeGRPCFrame(nil),
		}, nil
	}

	// Pipeline produced a response — encode JSON body to protobuf
	md, ok := g.methods[ref.GRPCPath]
	if !ok {
		return shared.TransformResponseResult{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/grpc",
			},
			Trailers: map[string]string{
				"Grpc-Status":  "12",
				"Grpc-Message": "method not found in proto: " + ref.GRPCPath,
			},
			Body: EncodeGRPCFrame(nil),
		}, nil
	}

	// Detect whether the response body is JSON or binary protobuf.
	// JSON responses are converted to protobuf; binary protobuf is used directly.
	var responseBytes []byte
	if json.Valid(args.ResponseBody) {
		var err error
		responseBytes, err = jsonToProto(args.ResponseBody, md.output)
		if err != nil {
			g.logger.Error("failed to marshal JSON response to protobuf", "path", ref.GRPCPath, "error", err)
			return shared.TransformResponseResult{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": "application/grpc",
				},
				Trailers: map[string]string{
					"Grpc-Status":  "13", // INTERNAL
					"Grpc-Message": "failed to marshal response: " + err.Error(),
				},
				Body: EncodeGRPCFrame(nil),
			}, nil
		}
	} else {
		g.logger.Debug("using binary protobuf response", "path", ref.GRPCPath)
		responseBytes = args.ResponseBody
	}

	return shared.TransformResponseResult{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/grpc",
		},
		Trailers: map[string]string{
			"Grpc-Status": grpcStatus,
		},
		Body: EncodeGRPCFrame(responseBytes),
	}, nil
}

func (g *GRPCPlugin) GenerateSyntheticData(_ shared.SyntheticDataRequest) (shared.SyntheticDataResponse, error) {
	return shared.SyntheticDataResponse{}, nil
}
