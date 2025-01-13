package soap

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWSDL2Operations(t *testing.T) {
	wsdlContent := `<?xml version="1.0" encoding="UTF-8"?>
<description xmlns="http://www.w3.org/ns/wsdl"
             xmlns:xsd="http://www.w3.org/2001/XMLSchema"
             xmlns:tns="http://example.com/test">

    <types>
        <xsd:schema targetNamespace="urn:com:example:petstore">
            <xsd:element name="TestRequest">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="id" type="xsd:int"/>
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>

            <xsd:element name="TestResponse">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="id" type="xsd:int"/>
                        <xsd:element name="name" type="xsd:string"/>
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>

            <xsd:element name="TestFault">
                <xsd:complexType>
                    <xsd:sequence>
                        <xsd:element name="code" type="xsd:string"/>
                        <xsd:element name="message" type="xsd:string"/>
                    </xsd:sequence>
                </xsd:complexType>
            </xsd:element>
        </xsd:schema>
    </types>
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
	assert.Equal(t, &xml.Name{Space: "http://www.w3.org/2001/XMLSchema", Local: "TestRequest"}, op.Input.Element)
	assert.Equal(t, &xml.Name{Space: "http://www.w3.org/2001/XMLSchema", Local: "TestResponse"}, op.Output.Element)
	assert.Equal(t, &xml.Name{Space: "http://www.w3.org/2001/XMLSchema", Local: "TestFault"}, op.Fault.Element)

	// Test GetBindingName
	assert.Equal(t, "TestBinding", parser.GetBindingName(op))
}
