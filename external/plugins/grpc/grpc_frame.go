package main

import (
	"encoding/binary"
	"fmt"
)

// gRPC uses a 5-byte header for length-prefixed messages:
// [1 byte: compressed flag] [4 bytes: big-endian message length] [N bytes: message]

const grpcFrameHeaderSize = 5

// DecodeGRPCFrame extracts the message bytes from a gRPC length-prefixed frame.
// Returns the raw protobuf message bytes.
func DecodeGRPCFrame(data []byte) ([]byte, error) {
	if len(data) < grpcFrameHeaderSize {
		return nil, fmt.Errorf("gRPC frame too short: %d bytes", len(data))
	}

	// First byte is compressed flag (0 = not compressed)
	compressed := data[0]
	if compressed != 0 {
		return nil, fmt.Errorf("compressed gRPC messages are not supported")
	}

	// Next 4 bytes are big-endian message length
	msgLen := binary.BigEndian.Uint32(data[1:5])

	if int(msgLen) > len(data)-grpcFrameHeaderSize {
		return nil, fmt.Errorf("gRPC frame declares %d bytes but only %d available", msgLen, len(data)-grpcFrameHeaderSize)
	}

	return data[grpcFrameHeaderSize : grpcFrameHeaderSize+int(msgLen)], nil
}

// EncodeGRPCFrame wraps a protobuf message in a gRPC length-prefixed frame.
func EncodeGRPCFrame(msg []byte) []byte {
	frame := make([]byte, grpcFrameHeaderSize+len(msg))
	frame[0] = 0 // not compressed
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(msg)))
	copy(frame[grpcFrameHeaderSize:], msg)
	return frame
}
