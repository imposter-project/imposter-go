package soap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWSDL2Operations(t *testing.T) {
	wsdlContent := `<?xml version="1.0" encoding="UTF-8"?>
<description xmlns="http://www.w3.org/ns/wsdl"
             xmlns:tns="http://example.com/test">
    <interface name="TestInterface">
        <operation name="TestOperation">
            <input messageLabel="In" element="tns:TestRequest"/>
            <output messageLabel="Out" element="tns:TestResponse"/>
            <outfault messageLabel="Fault" element="tns:TestFault"/>
        </operation>
    </interface>
    <binding name="TestBinding" interface="tns:TestInterface" type="http://www.w3.org/ns/wsdl/soap">
		<operation ref="tns:TestOperation"/>
	</binding>
</description>`

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

	// Test operation messages
	assert.Equal(t, "In", op.Input.Name)
	assert.Equal(t, "tns:TestRequest", op.Input.Element)
	assert.Equal(t, "Out", op.Output.Name)
	assert.Equal(t, "tns:TestResponse", op.Output.Element)
	assert.Equal(t, "Fault", op.Fault.Name)
	assert.Equal(t, "tns:TestFault", op.Fault.Element)

	// Test GetBindingName
	assert.Equal(t, "TestBinding", parser.GetBindingName(op))
}
