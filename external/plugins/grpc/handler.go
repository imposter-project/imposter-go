package main

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// jsonToProto converts JSON data to protobuf wire format using the given message descriptor.
func jsonToProto(jsonData []byte, msgDesc protoreflect.MessageDescriptor) ([]byte, error) {
	msg := dynamicpb.NewMessage(msgDesc)
	unmarshaler := protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}
	if err := unmarshaler.Unmarshal(jsonData, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to proto: %w", err)
	}
	return proto.Marshal(msg)
}
