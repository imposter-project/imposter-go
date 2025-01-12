package soap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWSDLParser(t *testing.T) {
	tests := []struct {
		name        string
		wsdlContent string
		wantVersion WSDLVersion
		wantSOAP    SOAPVersion
		wantErr     bool
	}{
		{
			name: "WSDL 1.1 with SOAP 1.1",
			wsdlContent: `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/test">
    <binding name="TestBinding" type="tns:TestPortType">
        <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="TestOperation">
            <soap:operation soapAction="http://example.com/test/action"/>
        </operation>
    </binding>
</definitions>`,
			wantVersion: WSDL1,
			wantSOAP:    SOAP11,
			wantErr:     false,
		},
		{
			name: "WSDL 1.1 with SOAP 1.2",
			wsdlContent: `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap12="http://schemas.xmlsoap.org/wsdl/soap12/"
             xmlns:tns="http://example.com/test">
    <binding name="TestBinding" type="tns:TestPortType">
        <soap12:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
        <operation name="TestOperation">
            <soap12:operation soapAction="http://example.com/test/action"/>
        </operation>
    </binding>
</definitions>`,
			wantVersion: WSDL1,
			wantSOAP:    SOAP12,
			wantErr:     false,
		},
		{
			name: "WSDL 2.0",
			wsdlContent: `<?xml version="1.0" encoding="UTF-8"?>
<description xmlns="http://www.w3.org/ns/wsdl"
             xmlns:tns="http://example.com/test">
    <interface name="TestInterface">
        <operation name="TestOperation">
            <input messageLabel="In" element="tns:TestRequest"/>
            <output messageLabel="Out" element="tns:TestResponse"/>
        </operation>
    </interface>
</description>`,
			wantVersion: WSDL2,
			wantSOAP:    SOAP12,
			wantErr:     false,
		},
		{
			name: "Invalid WSDL",
			wsdlContent: `<?xml version="1.0" encoding="UTF-8"?>
<invalid>
    <content>This is not a valid WSDL document</content>
</invalid>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary WSDL file
			tmpDir := t.TempDir()
			wsdlPath := filepath.Join(tmpDir, "test.wsdl")
			err := os.WriteFile(wsdlPath, []byte(tt.wsdlContent), 0644)
			require.NoError(t, err)

			// Parse WSDL
			parser, err := newWSDLParser(wsdlPath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, parser.GetVersion())
			assert.Equal(t, tt.wantSOAP, parser.GetSOAPVersion())
		})
	}
}

func TestValidateRequest(t *testing.T) {
	wsdlContent := `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:tns="http://example.com/test">
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

	// Test ValidateRequest (currently a no-op)
	err = parser.ValidateRequest("TestOperation", []byte("<test>data</test>"))
	assert.NoError(t, err)
}

func TestErrorCases(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		_, err := newWSDLParser("non_existent.wsdl")
		assert.Error(t, err)
	})

	t.Run("invalid XML", func(t *testing.T) {
		tmpDir := t.TempDir()
		wsdlPath := filepath.Join(tmpDir, "invalid.wsdl")
		err := os.WriteFile(wsdlPath, []byte("invalid xml content"), 0644)
		require.NoError(t, err)

		_, err = newWSDLParser(wsdlPath)
		assert.Error(t, err)
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		wsdlPath := filepath.Join(tmpDir, "empty.wsdl")
		err := os.WriteFile(wsdlPath, []byte(""), 0644)
		require.NoError(t, err)

		_, err = newWSDLParser(wsdlPath)
		assert.Error(t, err)
	})
}
