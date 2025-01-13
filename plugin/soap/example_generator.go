package soap

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/outofcoffee/go-xml-example-generator/examplegen"
)

// generateExampleXML generates example XML based on the WSDL schema
func generateExampleXML(element *xml.Name, parser *WSDLParser) (string, error) {
	var targetNS string

	//// Get target namespace from first schema (or root if not found)
	//for _, schemaNode := range schemas {
	//	if ns := schemaNode.SelectAttr("targetNamespace"); ns != "" {
	//		targetNS = ns
	//		break
	//	}
	//}
	//if targetNS == "" {
	//	// Try to get from root element as fallback
	//	if root := doc.SelectElement("*"); root != nil {
	//		targetNS = root.SelectAttr("targetNamespace")
	//	}
	//}

	schemas := (*((*parser).GetSchemaSystem())).GetSchemas()

	// Extract local part of element QName

	// localPart := getLocalPart(element)
	localPart := element.Local

	// prefix := getPrefix(element)
	prefix := "tns"

	elementExpr := fmt.Sprintf("//*[local-name()='element' and @name='%s']", localPart)

	// the path to the schema file that contains the element
	var elementSchemaPath string

	for _, schemaPath := range schemas {
		schemaContent, err := os.ReadFile(schemaPath)
		if err != nil {
			logger.Warnf("failed to read schema file %s: %v", schemaPath, err)
			continue
		}

		schemaDoc, err := xmlquery.Parse(strings.NewReader(string(schemaContent)))
		if err != nil {
			logger.Warnf("failed to parse schema file %s: %v", schemaPath, err)
			continue
		}

		elementNode := xmlquery.FindOne(schemaDoc, elementExpr)
		if elementNode != nil {
			// Found the element, remember the path to the schema file and get its target namespace
			elementSchemaPath = schemaPath

			if schemaRoot := schemaDoc.SelectElement("schema"); schemaRoot != nil {
				if ns := schemaRoot.SelectAttr("targetNamespace"); ns != "" {
					targetNS = ns
				}
			}
			break
		}
	}

	if elementSchemaPath == "" {
		return "", fmt.Errorf("element definition not found: %s (searched in WSDL and %d schema files)", element, len(schemas))
	}

	logger.Debugf("generating example for element [localPart: %s, prefix: %s, target namespace: %s]", localPart, prefix, targetNS)

	example, err := examplegen.GenerateWithNs(elementSchemaPath, localPart, targetNS, prefix)
	if err != nil {
		return "", fmt.Errorf("failed to generate XML: %w", err)
	}

	return example, nil
}
