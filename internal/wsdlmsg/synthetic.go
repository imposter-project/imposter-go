package wsdlmsg

import (
	"encoding/xml"
	"fmt"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/pkg/xsd"
	"path/filepath"
	"strings"
)

const (
	XMLSchemaNamespace = "http://www.w3.org/2001/XMLSchema"
	XMLSchemaNSPrefix  = "xs"
)

// CreateSinglePartSchema creates an XML schema for a single message part
func CreateSinglePartSchema(elementName string, elementType *xml.Name, targetNamespace string) (schema []byte, elementQName string) {
	logger.Tracef("creating single part schema for %s", elementName)

	var typeNsPrefix string
	if elementType.Space != XMLSchemaNamespace {
		typeNsPrefix = "ns1"
	} else {
		typeNsPrefix = XMLSchemaNSPrefix
	}

	elementQName = xsd.MakeQName(typeNsPrefix, elementName)

	// Build namespaces map
	namespaces := make(map[string]string)
	if elementType.Space != "" {
		namespaces[typeNsPrefix] = elementType.Space
	}
	namespaces[XMLSchemaNSPrefix] = XMLSchemaNamespace
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
func CreateCompositePartSchema(
	rootElementName string,
	parts []Message,
	targetNamespace string,
	imports []xsd.Schema,
) []byte {
	logger.Tracef("creating composite part schema for %s", rootElementName)

	// Build namespaces map
	namespaces := make(map[string]string)
	namespaces[XMLSchemaNSPrefix] = XMLSchemaNamespace
	if targetNamespace != "" {
		namespaces["tns"] = targetNamespace
	}

	// Generate imports XML
	var importsXml []string
	for _, schema := range imports {
		schemaLocation := filepath.Base(schema.FilePath)
		importsXml = append(importsXml, fmt.Sprintf(`    <xs:import namespace="%s" schemaLocation="%s"/>`, schema.TargetNamespace, schemaLocation))
	}

	var elements []string

	// Collect namespaces from all parts and generate complex type elements
	for _, part := range parts {
		switch m := part.(type) {
		case *ElementMessage:
			var nsPrefix string
			if m.Element.Space != "" {
				nsPrefix = getOrAddPrefixForNs(&namespaces, m.Element.Space)
			}
			qName := xsd.MakeQName(nsPrefix, m.Element.Local)
			elements = append(elements, fmt.Sprintf(`            <xs:element ref="%s"/>`, qName))

		case *TypeMessage:
			var nsPrefix string
			if m.Type.Space != "" {
				nsPrefix = getOrAddPrefixForNs(&namespaces, m.Type.Space)
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

%s

    <xs:element name="%s">
        <xs:complexType>
            <xs:sequence>
%s
            </xs:sequence>
        </xs:complexType>
    </xs:element>
</xs:schema>`, strings.Join(namespacesXml, "\n"), strings.Join(importsXml, "\n"), rootElementName, strings.Join(elements, "\n"))

	logger.Tracef("generated composite schema: %s", generatedSchema)
	return []byte(generatedSchema)
}

// getOrAddPrefixForNs returns the prefix for a namespace if it already exists, or adds a new one
// to the namespaces map and returns it
func getOrAddPrefixForNs(namespaces *map[string]string, ns string) string {
	if ns == XMLSchemaNamespace {
		// ns added by default to root schema element
		return XMLSchemaNSPrefix
	}

	for prefix, uri := range *namespaces {
		if uri == ns {
			return prefix
		}
	}

	nsPrefix := fmt.Sprintf("ns%d", len(*namespaces))
	(*namespaces)[nsPrefix] = ns
	return nsPrefix
}
