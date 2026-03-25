package fakedata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testProvider struct{}

func (p *testProvider) GenerateFakeData(req Request) Response {
	if req.ExprCategory == "Name" && req.ExprProperty == "firstName" {
		return Response{Value: "TestName", Found: true}
	}
	if req.PropertyName == "email" {
		return Response{Value: "test@test.com", Found: true}
	}
	if req.Format == "email" {
		return Response{Value: "format@test.com", Found: true}
	}
	return Response{}
}

func TestRegisterAndGetProvider(t *testing.T) {
	// Clean state
	RegisterProvider(nil)
	assert.Nil(t, GetProvider())

	// Register
	p := &testProvider{}
	RegisterProvider(p)
	assert.NotNil(t, GetProvider())

	// Cleanup
	RegisterProvider(nil)
	assert.Nil(t, GetProvider())
}

func TestGenerate_WithProvider(t *testing.T) {
	RegisterProvider(&testProvider{})
	defer RegisterProvider(nil)

	val := Generate("Name", "firstName")
	assert.Equal(t, "TestName", val)
}

func TestGenerate_WithoutProvider(t *testing.T) {
	RegisterProvider(nil)

	val := Generate("Name", "firstName")
	assert.Equal(t, "", val)
}

func TestGenerate_NotFound(t *testing.T) {
	RegisterProvider(&testProvider{})
	defer RegisterProvider(nil)

	val := Generate("Unknown", "thing")
	assert.Equal(t, "", val)
}

func TestGenerateForPropertyName_WithProvider(t *testing.T) {
	RegisterProvider(&testProvider{})
	defer RegisterProvider(nil)

	val, ok := GenerateForPropertyName("email")
	assert.True(t, ok)
	assert.Equal(t, "test@test.com", val)
}

func TestGenerateForPropertyName_WithoutProvider(t *testing.T) {
	RegisterProvider(nil)

	val, ok := GenerateForPropertyName("email")
	assert.False(t, ok)
	assert.Equal(t, "", val)
}

func TestGenerateForFormat_WithProvider(t *testing.T) {
	RegisterProvider(&testProvider{})
	defer RegisterProvider(nil)

	val, ok := GenerateForFormat("email")
	assert.True(t, ok)
	assert.Equal(t, "format@test.com", val)
}

func TestGenerateForFormat_WithoutProvider(t *testing.T) {
	RegisterProvider(nil)

	val, ok := GenerateForFormat("email")
	assert.False(t, ok)
	assert.Equal(t, "", val)
}
