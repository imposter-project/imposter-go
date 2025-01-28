package wsdlmsg

import (
	"encoding/xml"
	"fmt"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/pkg/xsd"
	"strings"
)

const (
	XMLSchemaNamespace = "http://www.w3.org/2001/XMLSchema"
)

// CreateSinglePartSchema creates an XML schema for a single message part
func CreateSinglePartSchema(elementName string, elementType *xml.Name, targetNamespace string) (schema []byte, elementQName string) {
	var typeNsPrefix string
	if elementType.Space != XMLSchemaNamespace {
		typeNsPrefix = "ns1"
	} else {
		typeNsPrefix = "xs"
	}

	elementQName = xsd.MakeQName(typeNsPrefix, elementName)

	// Build namespaces map
	namespaces := make(map[string]string)
	if elementType.Space != "" {
		namespaces[typeNsPrefix] = elementType.Space
	}
	namespaces["xs"] = XMLSchemaNamespace
	if targetNamespace != "" {
		namespaces["tns"] = targetNamespace
	}

	// Generate namespaces XML
	var namespacesXml []string
	for prefix, uri := range namespaces {
		namespacesXml = append(namespacesXml, fmt.Sprintf(`xmlns:%s="%s"`, prefix, uri))
	}

	// Add targetNamespace attribute if provided
	if targetNamespace != "" {
		namespacesXml = append(namespacesXml, fmt.Sprintf(`targetNamespace="%s"`, targetNamespace))
	}

	typeQName := xsd.MakeQName(typeNsPrefix, elementType.Local)

	// Generate element XML
	elementXml := fmt.Sprintf(`<xs:element name="%s" type="%s"/>`, elementName, typeQName)

	// Build complete schema
	generatedSchema := fmt.Sprintf(`<xs:schema elementFormDefault="qualified" version="1.0"
%s>

%s
</xs:schema>`, strings.Join(namespacesXml, "\n"), elementXml)

	logger.Tracef("generated single part schema: %s", generatedSchema)
	return []byte(generatedSchema), elementQName
}

// CreateCompositePartSchema creates an XML schema for a composite message part
func CreateCompositePartSchema(rootElementName string, parts []Message, targetNamespace string) []byte {
	// Build namespaces map
	namespaces := make(map[string]string)
	namespaces["xs"] = XMLSchemaNamespace
	if targetNamespace != "" {
		namespaces["tns"] = targetNamespace
	}

	nsIndex := 0
	var elements []string

	// Collect namespaces from all parts and generate complex type elements
	for _, part := range parts {
		nsIndex++

		switch m := part.(type) {
		case *ElementMessage:
			var nsPrefix string
			if m.Element.Space != "" {
				if m.Element.Space != XMLSchemaNamespace {
					nsPrefix = getOrAddPrefixForNs(&namespaces, m.Element.Space)
				} else {
					nsPrefix = "xs"
				}
			}
			qName := xsd.MakeQName(nsPrefix, m.Element.Local)
			elements = append(elements, fmt.Sprintf(`            <xs:element ref="%s"/>`, qName))

		case *TypeMessage:
			var nsPrefix string
			if m.Type.Space != "" {
				if m.Type.Space != XMLSchemaNamespace {
					nsPrefix = getOrAddPrefixForNs(&namespaces, m.Type.Space)
				} else {
					nsPrefix = "xs"
				}
			}
			qName := xsd.MakeQName(nsPrefix, m.Type.Local)
			elements = append(elements, fmt.Sprintf(`            <xs:element name="%s" type="%s"/>`, m.PartName, qName))
		}
	}

	// Generate namespaces XML
	var namespacesXml []string
	for prefix, uri := range namespaces {
		namespacesXml = append(namespacesXml, fmt.Sprintf(`xmlns:%s="%s"`, prefix, uri))
	}

	// Add targetNamespace attribute if provided
	if targetNamespace != "" {
		namespacesXml = append(namespacesXml, fmt.Sprintf(`targetNamespace="%s"`, targetNamespace))
	}

	// Build complete schema with complex type
	generatedSchema := fmt.Sprintf(`<xs:schema elementFormDefault="qualified" version="1.0"
%s>

    <xs:element name="%s">
        <xs:complexType>
            <xs:sequence>
%s
            </xs:sequence>
        </xs:complexType>
    </xs:element>
</xs:schema>`, strings.Join(namespacesXml, "\n"), rootElementName, strings.Join(elements, "\n"))

	logger.Tracef("generated composite schema: %s", generatedSchema)
	return []byte(generatedSchema)
}

// getOrAddPrefixForNs returns the prefix for a namespace if it already exists, or adds a new one
// to the namespaces map and returns it
func getOrAddPrefixForNs(namespaces *map[string]string, ns string) string {
	for prefix, uri := range *namespaces {
		if uri == ns {
			return prefix
		}
	}

	nsPrefix := fmt.Sprintf("ns%d", len(*namespaces))
	(*namespaces)[nsPrefix] = ns
	return nsPrefix
}
