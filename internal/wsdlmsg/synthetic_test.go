package wsdlmsg

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestCreateSinglePartSchema(t *testing.T) {
	tests := []struct {
		name            string
		message         *TypeMessage
		targetNamespace string
		want            []string // substrings that should be present in result
	}{
		{
			name: "Simple type with no target namespace",
			message: &TypeMessage{
				PartName: "body",
				Type: &xml.Name{
					Space: "tns",
					Local: "MyType",
				},
			},
			targetNamespace: "",
			want: []string{
				`xmlns:xs="http://www.w3.org/2001/XMLSchema"`,
				`xmlns:tns="tns"`,
				`<xs:element name="body" type="tns:MyType"/>`,
			},
		},
		{
			name: "Type with target namespace",
			message: &TypeMessage{
				PartName: "request",
				Type: &xml.Name{
					Space: "ns1",
					Local: "RequestType",
				},
			},
			targetNamespace: "http://example.org/schema",
			want: []string{
				`xmlns:xs="http://www.w3.org/2001/XMLSchema"`,
				`xmlns:ns1="ns1"`,
				`xmlns:tns="http://example.org/schema"`,
				`targetNamespace="http://example.org/schema"`,
				`<xs:element name="request" type="ns1:RequestType"/>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSchema, gotElementName := CreateSinglePartSchema(tt.message, tt.targetNamespace)
			if gotElementName != tt.message.PartName {
				t.Errorf("CreateSinglePartSchema() element name = %v, want %v", gotElementName, tt.message.PartName)
			}
			for _, want := range tt.want {
				if !strings.Contains(string(gotSchema), want) {
					t.Errorf("CreateSinglePartSchema() = %v, should contain %v", gotSchema, want)
				}
			}
		})
	}
}

func TestCreateCompositePartSchema(t *testing.T) {
	tests := []struct {
		name            string
		rootElementName string
		parts           []Message
		targetNamespace string
		want            []string
	}{
		{
			name:            "Mixed element and type parts",
			rootElementName: "getPetResponse",
			parts: []Message{
				&ElementMessage{
					Element: &xml.Name{
						Space: "pet",
						Local: "Pet",
					},
				},
				&TypeMessage{
					PartName: "status",
					Type: &xml.Name{
						Space: "ns1",
						Local: "StatusType",
					},
				},
			},
			targetNamespace: "http://example.org/pets",
			want: []string{
				`xmlns:xs="http://www.w3.org/2001/XMLSchema"`,
				`xmlns:pet="pet"`,
				`xmlns:ns1="ns1"`,
				`xmlns:tns="http://example.org/pets"`,
				`targetNamespace="http://example.org/pets"`,
				`<xs:element name="getPetResponse">`,
				`<xs:complexType>`,
				`<xs:sequence>`,
				`<xs:element ref="pet:Pet"/>`,
				`<xs:element name="status" type="ns1:StatusType"/>`,
			},
		},
		{
			name:            "Only type messages",
			rootElementName: "userRequest",
			parts: []Message{
				&TypeMessage{
					PartName: "id",
					Type: &xml.Name{
						Space: "xs",
						Local: "string",
					},
				},
				&TypeMessage{
					PartName: "name",
					Type: &xml.Name{
						Space: "xs",
						Local: "string",
					},
				},
			},
			targetNamespace: "",
			want: []string{
				`xmlns:xs="http://www.w3.org/2001/XMLSchema"`,
				`<xs:element name="userRequest">`,
				`<xs:complexType>`,
				`<xs:sequence>`,
				`<xs:element name="id" type="xs:string"/>`,
				`<xs:element name="name" type="xs:string"/>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateCompositePartSchema(tt.rootElementName, tt.parts, tt.targetNamespace)
			for _, want := range tt.want {
				if !strings.Contains(string(got), want) {
					t.Errorf("CreateCompositePartSchema() = %v, should contain %v", got, want)
				}
			}
		})
	}
}
