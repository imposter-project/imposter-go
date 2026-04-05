package soap

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/pkg/logger"
	wsdlparser "github.com/outofcoffee/go-wsdl-parser"
	"github.com/outofcoffee/go-wsdl-parser/wsdlmsg"
	"github.com/outofcoffee/go-wsdl-parser/xsd"
	"github.com/outofcoffee/go-xml-example-generator/examplegen"
)

// generateExampleXML generates example XML for a WSDL message.
//
// For document-literal element messages, the existing element lookup path
// across the loaded schemas is used. For RPC-style operations, whose messages
// carry type-references rather than element-references, a synthetic wrapper
// schema is compiled that declares an element named after the operation
// (plus the "Response" suffix for output messages), containing the message
// parts as child elements — mirroring the approach taken by the JVM engine.
func generateExampleXML(
	op *wsdlparser.Operation,
	message *wsdlmsg.Message,
	schemaSystem *xsd.SchemaSystem,
	targetNamespace string,
	isResponse bool,
) (string, error) {
	// TODO cache example responses

	if message == nil || *message == nil {
		return "", fmt.Errorf("no message to generate example for")
	}

	// For RPC-style operations the SOAP body contains a wrapper element
	// named after the operation (with "Response" appended for responses).
	// Document-style operations use the element or message name directly.
	var wrapperName string
	if op != nil && strings.EqualFold(op.Style, wsdlparser.StyleRPC) {
		wrapperName = op.Name
		if isResponse {
			wrapperName += "Response"
		}
	}

	switch (*message).GetMessageType() {
	case wsdlmsg.ElementMessageType:
		elementMsg := (*message).(*wsdlmsg.ElementMessage)
		if wrapperName == "" {
			return generateFromExistingElement(elementMsg.Element, schemaSystem)
		}
		// RPC operation referring to an element: still wrap under the
		// operation name, with the element declared as a reference.
		return generateFromSyntheticSchema(wrapperName, []wsdlmsg.Message{elementMsg}, schemaSystem, targetNamespace)

	case wsdlmsg.TypeMessageType:
		typeMsg := (*message).(*wsdlmsg.TypeMessage)
		if wrapperName == "" {
			// Document-style messages rarely use type refs, but handle the
			// case by declaring a single top-level element for the part.
			wrapperName = typeMsg.PartName
		}
		return generateFromSyntheticSchema(wrapperName, []wsdlmsg.Message{typeMsg}, schemaSystem, targetNamespace)

	case wsdlmsg.CompositeMessageType:
		compositeMsg := (*message).(*wsdlmsg.CompositeMessage)
		if wrapperName == "" {
			wrapperName = compositeMsg.MessageName
		}
		var parts []wsdlmsg.Message
		if compositeMsg.Parts != nil {
			parts = *compositeMsg.Parts
		}
		return generateFromSyntheticSchema(wrapperName, parts, schemaSystem, targetNamespace)

	default:
		return "", fmt.Errorf("unsupported message type: %T", *message)
	}
}

// generateFromExistingElement generates an example for an element that is
// already defined in one of the WSDL's XSD schemas.
func generateFromExistingElement(element *xml.Name, schemaSystem *xsd.SchemaSystem) (string, error) {
	localPart := element.Local
	elementExpr := fmt.Sprintf("//*[local-name()='element' and @name='%s']", localPart)

	var elementSchemaPath string
	var targetNS string

	schemas := (*schemaSystem).GetSchemas()
	for _, schema := range schemas {
		schemaContent, err := os.ReadFile(schema.FilePath)
		if err != nil {
			logger.Warnf("failed to read schema file %s: %v", schema.FilePath, err)
			continue
		}

		schemaDoc, err := xmlquery.Parse(strings.NewReader(string(schemaContent)))
		if err != nil {
			logger.Warnf("failed to parse schema file %s: %v", schema.FilePath, err)
			continue
		}

		elementNode := xmlquery.FindOne(schemaDoc, elementExpr)
		if elementNode != nil {
			elementSchemaPath = schema.FilePath
			ns := xsd.GetTargetNamespace(schemaDoc)
			if ns != "" {
				targetNS = ns
			}
			break
		}
	}

	if elementSchemaPath == "" {
		return "", fmt.Errorf("element definition not found: %s (searched in WSDL and %d schema files)", element, len(schemas))
	}

	logger.Debugf("generating example for element [localPart: %s, target namespace: %s]", localPart, targetNS)

	example, err := examplegen.GenerateWithNs(elementSchemaPath, localPart, targetNS, "tns")
	if err != nil {
		return "", fmt.Errorf("failed to generate XML: %w", err)
	}
	return example, nil
}

// generateFromSyntheticSchema builds an XSD schema on the fly that declares
// a wrapper element containing the given parts, places it alongside the
// existing extracted schemas so that relative xs:import schemaLocations
// continue to resolve, and asks the example generator to produce a sample.
func generateFromSyntheticSchema(
	wrapperName string,
	parts []wsdlmsg.Message,
	schemaSystem *xsd.SchemaSystem,
	targetNamespace string,
) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("no parts provided for wrapper element %q", wrapperName)
	}

	// Collect the existing schemas so that the synthetic schema can import
	// them for any referenced types or elements.
	schemas := (*schemaSystem).GetSchemas()
	var imports []xsd.Schema
	var schemaDir string
	for _, s := range schemas {
		imports = append(imports, s)
		if schemaDir == "" {
			schemaDir = filepath.Dir(s.FilePath)
		}
	}
	if schemaDir == "" {
		// Fall back to a fresh temp directory if no schemas were extracted.
		var err error
		schemaDir, err = os.MkdirTemp("", "imposter-synth-schema-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temp schema dir: %w", err)
		}
	}

	schemaBytes := wsdlmsg.CreateCompositePartSchema(wrapperName, parts, targetNamespace, imports)

	syntheticPath := filepath.Join(schemaDir, fmt.Sprintf("imposter-synth-%s.xsd", wrapperName))
	if err := os.WriteFile(syntheticPath, schemaBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write synthetic schema: %w", err)
	}
	defer func() {
		if err := os.Remove(syntheticPath); err != nil {
			logger.Debugf("failed to remove synthetic schema %s: %v", syntheticPath, err)
		}
	}()

	logger.Debugf("generating example from synthetic schema [wrapper: %s, target namespace: %s, parts: %d]",
		wrapperName, targetNamespace, len(parts))

	example, err := examplegen.GenerateWithNs(syntheticPath, wrapperName, targetNamespace, "tns")
	if err != nil {
		return "", fmt.Errorf("failed to generate XML from synthetic schema for %q: %w", wrapperName, err)
	}
	return example, nil
}
