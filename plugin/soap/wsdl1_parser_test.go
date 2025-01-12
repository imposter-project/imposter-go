package soap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWSDL1Operations(t *testing.T) {
	wsdlContent := `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/test">
    <binding name="TestBinding" type="tns:TestPortType">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="TestOperation">
            <soap:operation soapAction="http://example.com/test/action"/>
            <input name="TestInput" element="tns:TestRequest"/>
            <output name="TestOutput" element="tns:TestResponse"/>
            <fault name="TestFault" element="tns:TestFault"/>
        </operation>
    </binding>
</definitions>`

	// Create temporary WSDL file
	tmpDir := t.TempDir()
	wsdlPath := filepath.Join(tmpDir, "test.wsdl")
	err := os.WriteFile(wsdlPath, []byte(wsdlContent), 0644)
	require.NoError(t, err)

	// Parse WSDL
	parser, err := newWSDLParser(wsdlPath)
	require.NoError(t, err)

	// Test GetOperations
	ops := parser.GetOperations()
	require.Len(t, ops, 1)

	// Test GetOperation
	op := parser.GetOperation("TestOperation")
	require.NotNil(t, op)
	assert.Equal(t, "TestOperation", op.Name)
	assert.Equal(t, "http://example.com/test/action", op.SOAPAction)
	assert.Equal(t, "TestBinding", op.Binding)

	// Test operation messages
	assert.Equal(t, "TestInput", op.Input.Name)
	assert.Equal(t, "tns:TestRequest", op.Input.Element)
	assert.Equal(t, "TestOutput", op.Output.Name)
	assert.Equal(t, "tns:TestResponse", op.Output.Element)
	assert.Equal(t, "TestFault", op.Fault.Name)
	assert.Equal(t, "tns:TestFault", op.Fault.Element)

	// Test GetBindingName
	assert.Equal(t, "TestBinding", parser.GetBindingName(op))
}
