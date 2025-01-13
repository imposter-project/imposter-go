package soap

import (
	"encoding/xml"
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

	<message name="TestInput">
        <part name="parameters" element="tns:TestRequest"/>
    </message>
    <message name="TestOutput">
        <part name="parameters" element="tns:TestResponse"/>
    </message>
    <message name="TestFault">
        <part name="parameters" element="tns:TestFault"/>
    </message>
    <portType name="TestPortType">
        <operation name="TestOperation">
            <input message="TestInput"/>
            <output message="TestOutput"/>
            <fault message="TestFault"/>
        </operation>
    </portType>
    <binding name="TestBinding" type="tns:TestPortType">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="TestOperation">
            <soap:operation soapAction="http://example.com/test/action"/>
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
	assert.Equal(t, &xml.Name{Space: "http://www.w3.org/2001/XMLSchema", Local: "TestRequest"}, op.Input.Element)
	assert.Equal(t, &xml.Name{Space: "http://www.w3.org/2001/XMLSchema", Local: "TestResponse"}, op.Output.Element)
	assert.Equal(t, &xml.Name{Space: "http://www.w3.org/2001/XMLSchema", Local: "TestFault"}, op.Fault.Element)

	// Test GetBindingName
	assert.Equal(t, "TestBinding", parser.GetBindingName(op))
}
