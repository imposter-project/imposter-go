package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/external/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func testLogger() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Off,
		Output: os.Stderr,
	})
}

func testConfigDir() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata")
}

func newTestPlugin(t *testing.T) *GRPCPlugin {
	t.Helper()
	p := &GRPCPlugin{logger: testLogger()}
	cfg := shared.ExternalConfig{
		Configs: []shared.LightweightConfig{
			{
				ConfigDir:    testConfigDir(),
				Plugin:       "grpc",
				PluginConfig: []byte("protoFiles:\n  - test.proto"),
			},
		},
	}
	caps, err := p.Configure(cfg)
	require.NoError(t, err)
	assert.True(t, caps.HandleRequests)
	return p
}

func TestConfigure(t *testing.T) {
	p := newTestPlugin(t)
	assert.Contains(t, p.methods, "/test.Greeter/SayHello")
}

func TestConfigure_MissingProtoFile(t *testing.T) {
	p := &GRPCPlugin{logger: testLogger()}
	cfg := shared.ExternalConfig{
		Configs: []shared.LightweightConfig{
			{
				ConfigDir:    testConfigDir(),
				Plugin:       "grpc",
				PluginConfig: []byte("protoFiles:\n  - nonexistent.proto"),
			},
		},
	}
	_, err := p.Configure(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent.proto")
}

func TestNormaliseRequest_NonGRPC(t *testing.T) {
	p := newTestPlugin(t)
	resp, _ := p.NormaliseRequest(shared.HandlerRequest{
		Method:  "POST",
		Path:    "/test.Greeter/SayHello",
		Headers: map[string]string{"Content-Type": "application/json"},
	})
	assert.True(t, resp.Skip, "non-gRPC requests should be skipped")
}

func TestNormaliseRequest_GETSkipped(t *testing.T) {
	p := newTestPlugin(t)
	resp, _ := p.NormaliseRequest(shared.HandlerRequest{
		Method:  "GET",
		Path:    "/test.Greeter/SayHello",
		Headers: map[string]string{"Content-Type": "application/grpc"},
	})
	assert.True(t, resp.Skip, "gRPC requires POST")
}

func TestNormaliseRequest_DecodesProtobufToJSON(t *testing.T) {
	p := newTestPlugin(t)

	// Build a protobuf-encoded HelloRequest with name = "World"
	md := p.methods["/test.Greeter/SayHello"]
	require.NotNil(t, md)
	msg := dynamicpb.NewMessage(md.input)
	msg.Set(md.input.Fields().ByName("name"), protoreflect.ValueOfString("World"))
	protoBytes, err := proto.Marshal(msg)
	require.NoError(t, err)

	resp, _ := p.NormaliseRequest(shared.HandlerRequest{
		Method:  "POST",
		Path:    "/test.Greeter/SayHello",
		Headers: map[string]string{"Content-Type": "application/grpc"},
		Body:    EncodeGRPCFrame(protoBytes),
	})

	assert.False(t, resp.Skip)
	assert.NotEmpty(t, resp.Body, "body should be decoded to JSON")
	assert.NotEmpty(t, resp.Metadata)

	// Verify the JSON body contains the decoded field
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body, &body))
	assert.Equal(t, "World", body["name"])
}

func TestNormaliseRequest_UnknownMethodStillAccepted(t *testing.T) {
	p := newTestPlugin(t)
	resp, _ := p.NormaliseRequest(shared.HandlerRequest{
		Method:  "POST",
		Path:    "/test.Greeter/UnknownMethod",
		Headers: map[string]string{"Content-Type": "application/grpc"},
		Body:    EncodeGRPCFrame(nil),
	})
	// Unknown methods are accepted but will get UNIMPLEMENTED in TransformResponse
	assert.False(t, resp.Skip)
	assert.NotEmpty(t, resp.Metadata)
}

func TestTransformResponse_PipelineHandled(t *testing.T) {
	p := newTestPlugin(t)

	metadata, _ := json.Marshal(methodRef{GRPCPath: "/test.Greeter/SayHello"})
	result, _ := p.TransformResponse(shared.TransformRequest{
		Handled:      true,
		ResponseBody: []byte(`{"message": "Hello, World!"}`),
		Metadata:     metadata,
	})

	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, "application/grpc", result.Headers["Content-Type"])
	assert.Equal(t, "0", result.Trailers["Grpc-Status"])
	assert.NotContains(t, result.Headers, "Grpc-Status", "Grpc-Status must be a trailer, not a header")

	// Decode and verify the protobuf response
	protoBytes, err := DecodeGRPCFrame(result.Body)
	require.NoError(t, err)

	md := p.methods["/test.Greeter/SayHello"]
	msg := dynamicpb.NewMessage(md.output)
	require.NoError(t, proto.Unmarshal(protoBytes, msg))

	jsonBytes, _ := protojson.Marshal(msg)
	var body map[string]interface{}
	json.Unmarshal(jsonBytes, &body)
	assert.Equal(t, "Hello, World!", body["message"])
}

func TestTransformResponse_PipelineNotHandled(t *testing.T) {
	p := newTestPlugin(t)

	metadata, _ := json.Marshal(methodRef{GRPCPath: "/test.Greeter/SayHello"})
	result, _ := p.TransformResponse(shared.TransformRequest{
		Handled:  false,
		Metadata: metadata,
	})

	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, "12", result.Trailers["Grpc-Status"]) // UNIMPLEMENTED
}

func TestTransformResponse_CustomGRPCStatus(t *testing.T) {
	p := newTestPlugin(t)

	metadata, _ := json.Marshal(methodRef{GRPCPath: "/test.Greeter/SayHello"})
	result, _ := p.TransformResponse(shared.TransformRequest{
		Handled:      true,
		StatusCode:   5, // NOT_FOUND
		ResponseBody: []byte(`{"message": ""}`),
		Metadata:     metadata,
	})

	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, "5", result.Trailers["Grpc-Status"])
}

func TestTransformResponse_DefaultHTTPStatusMapsToGRPCOK(t *testing.T) {
	p := newTestPlugin(t)

	metadata, _ := json.Marshal(methodRef{GRPCPath: "/test.Greeter/SayHello"})
	result, _ := p.TransformResponse(shared.TransformRequest{
		Handled:      true,
		StatusCode:   200, // default HTTP status from pipeline
		ResponseBody: []byte(`{"message": "Hello!"}`),
		Metadata:     metadata,
	})

	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, "0", result.Trailers["Grpc-Status"], "HTTP 200 should map to gRPC OK (0)")
}

func TestTransformResponse_CustomGRPCStatusNoBody(t *testing.T) {
	p := newTestPlugin(t)

	metadata, _ := json.Marshal(methodRef{GRPCPath: "/test.Greeter/SayHello"})
	result, _ := p.TransformResponse(shared.TransformRequest{
		Handled:    true,
		StatusCode: 5, // NOT_FOUND
		Metadata:   metadata,
		// No ResponseBody — typical for error status responses
	})

	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, "5", result.Trailers["Grpc-Status"])
	assert.Empty(t, result.Trailers["Grpc-Message"], "no error message expected")
}

func TestTransformResponse_NonJSONTreatedAsBinaryProtobuf(t *testing.T) {
	p := newTestPlugin(t)

	// Non-JSON data is treated as binary protobuf and used directly
	binaryData := []byte{0x0a, 0x05, 0x48, 0x65, 0x6c, 0x6c, 0x6f} // protobuf: field 1, string "Hello"
	metadata, _ := json.Marshal(methodRef{GRPCPath: "/test.Greeter/SayHello"})
	result, _ := p.TransformResponse(shared.TransformRequest{
		Handled:      true,
		ResponseBody: binaryData,
		Metadata:     metadata,
	})

	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, "0", result.Trailers["Grpc-Status"])

	decoded, err := DecodeGRPCFrame(result.Body)
	require.NoError(t, err)
	assert.Equal(t, binaryData, decoded)
}

func TestJsonToProto_RoundTrip(t *testing.T) {
	p := newTestPlugin(t)
	md := p.methods["/test.Greeter/SayHello"]
	require.NotNil(t, md)

	original := `{"message":"round trip test"}`
	protoBytes, err := jsonToProto([]byte(original), md.output)
	require.NoError(t, err)

	msg := dynamicpb.NewMessage(md.output)
	require.NoError(t, proto.Unmarshal(protoBytes, msg))

	jsonBytes, _ := protojson.Marshal(msg)
	assert.Contains(t, string(jsonBytes), "round trip test")
}

func TestTransformResponse_BinaryProtobuf(t *testing.T) {
	p := newTestPlugin(t)
	md := p.methods["/test.Greeter/SayHello"]
	require.NotNil(t, md)

	// Build binary protobuf directly (not JSON)
	msg := dynamicpb.NewMessage(md.output)
	msg.Set(md.output.Fields().ByName("message"), protoreflect.ValueOfString("binary hello"))
	protoBytes, err := proto.Marshal(msg)
	require.NoError(t, err)

	metadata, _ := json.Marshal(methodRef{GRPCPath: "/test.Greeter/SayHello"})
	result, _ := p.TransformResponse(shared.TransformRequest{
		Handled:      true,
		ResponseBody: protoBytes, // raw protobuf, not JSON
		Metadata:     metadata,
	})

	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, "0", result.Trailers["Grpc-Status"])

	// Decode the gRPC frame and verify the protobuf is used directly
	decoded, err := DecodeGRPCFrame(result.Body)
	require.NoError(t, err)
	assert.Equal(t, protoBytes, decoded, "binary protobuf should be used as-is")
}

func TestLoadGRPCConfig(t *testing.T) {
	config, err := loadGRPCConfig([]byte("protoFiles:\n  - test.proto\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"test.proto"}, config.ProtoFiles)
}

func TestLoadGRPCConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{name: "empty", yaml: "", wantErr: "requires configuration"},
		{name: "no proto files", yaml: "protoFiles: []", wantErr: "proto file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadGRPCConfig([]byte(tt.yaml))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
