package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeGRPCFrame(t *testing.T) {
	msg := []byte{0x08, 0x01} // protobuf: field 1, varint 1
	frame := EncodeGRPCFrame(msg)

	assert.Equal(t, byte(0), frame[0], "compression flag should be 0")
	assert.Equal(t, []byte{0, 0, 0, 2}, frame[1:5], "message length should be 2")
	assert.Equal(t, msg, frame[5:], "message bytes should follow header")
}

func TestEncodeGRPCFrame_Empty(t *testing.T) {
	frame := EncodeGRPCFrame(nil)

	assert.Len(t, frame, 5)
	assert.Equal(t, byte(0), frame[0])
	assert.Equal(t, []byte{0, 0, 0, 0}, frame[1:5])
}

func TestDecodeGRPCFrame(t *testing.T) {
	msg := []byte{0x08, 0x01}
	frame := EncodeGRPCFrame(msg)

	decoded, err := DecodeGRPCFrame(frame)
	require.NoError(t, err)
	assert.Equal(t, msg, decoded)
}

func TestDecodeGRPCFrame_TooShort(t *testing.T) {
	_, err := DecodeGRPCFrame([]byte{0, 0})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestDecodeGRPCFrame_Compressed(t *testing.T) {
	frame := []byte{1, 0, 0, 0, 1, 0x08}
	_, err := DecodeGRPCFrame(frame)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compressed")
}

func TestDecodeGRPCFrame_LengthMismatch(t *testing.T) {
	// Header says 10 bytes but only 2 available
	frame := []byte{0, 0, 0, 0, 10, 0x08, 0x01}
	_, err := DecodeGRPCFrame(frame)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "available")
}

func TestRoundTrip(t *testing.T) {
	original := []byte("hello world")
	frame := EncodeGRPCFrame(original)
	decoded, err := DecodeGRPCFrame(frame)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}
