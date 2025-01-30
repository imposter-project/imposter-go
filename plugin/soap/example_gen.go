package soap

import (
	"encoding/xml"
	"fmt"
	"github.com/imposter-project/imposter-go/internal/wsdlmsg"
	"github.com/imposter-project/imposter-go/pkg/xsd"
	"os"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/outofcoffee/go-xml-example-generator/examplegen"
)

const (
	soapExamplePlaceholder = "${soap.example()}"
)

// generateExampleXML generates example XML based on the WSDL schema
func generateExampleXML(message *wsdlmsg.Message, schemaSystem *xsd.SchemaSystem) (string, error) {
	// TODO cache example responses for each message

	var element *xml.Name
	switch (*message).GetMessageType() {
	case wsdlmsg.ElementMessageType:
		element = (*message).(*wsdlmsg.ElementMessage).Element
	default:
		return "", fmt.Errorf("unsupported message type: %T", *message)
	}

	localPart := element.Local
	elementExpr := fmt.Sprintf("//*[local-name()='element' and @name='%s']", localPart)

	// the path to the schema file that contains the element
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
			// Found the element, remember the path to the schema file and get its target namespace
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
