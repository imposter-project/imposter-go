package soap // WSDL 1.1 Parser

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/internal/wsdlmsg"
	"github.com/imposter-project/imposter-go/pkg/utils"
	"github.com/imposter-project/imposter-go/pkg/xsd"
	"strings"

	"github.com/antchfx/xmlquery"
)

type wsdl1Parser struct {
	BaseWSDLParser
}

func newWSDL1Parser(doc *xmlquery.Node, wsdlPath string) (*wsdl1Parser, error) {
	schemas, err := xsd.ExtractSchemas(wsdlPath, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schemas: %w", err)
	}
	targetNamespace := xsd.GetTargetNamespace(doc)
	parser := &wsdl1Parser{
		BaseWSDLParser: BaseWSDLParser{
			doc:             doc,
			wsdlPath:        wsdlPath,
			operations:      make(map[string]*Operation),
			schemas:         &schemas,
			targetNamespace: targetNamespace,
		},
	}
	if err := parser.parseOperations(); err != nil {
		return nil, err
	}
	if err := parser.resolveMessagesToElements(); err != nil {
		return nil, err
	}
	return parser, nil
}

func (p *wsdl1Parser) GetVersion() WSDLVersion {
	return WSDL1
}

func (p *wsdl1Parser) GetSOAPVersion() SOAPVersion {
	// Check for SOAP 1.2 binding
	if node := xmlquery.FindOne(p.doc, "//soap12:binding"); node != nil {
		return SOAP12
	}
	return SOAP11
}

func (p *wsdl1Parser) GetOperations() map[string]*Operation {
	return p.operations
}

func (p *wsdl1Parser) GetOperation(name string) *Operation {
	return p.operations[name]
}

func (p *wsdl1Parser) ValidateRequest(operation string, body []byte) error {
	// TODO: Implement schema validation
	return nil
}

func (p *wsdl1Parser) parseOperations() error {
	// First try to find operations in portType elements
	portTypeNodes := xmlquery.Find(p.doc, "//wsdl:portType|//portType")
	if len(portTypeNodes) > 0 {
		for _, portType := range portTypeNodes {
			portTypeName := portType.SelectAttr("name")
			// Find all operation nodes within this portType
			operationNodes := xmlquery.Find(portType, "./wsdl:operation|./operation")
			for _, node := range operationNodes {
				if err := p.parseOperation(node, portTypeName); err != nil {
					return err
				}
			}
		}
	} else {
		// TODO check if we need to support this for WSDL 1
		//// If no portType found, try to get operations from binding elements
		//bindingNodes := xmlquery.Find(p.doc, "//wsdl:binding|//binding")
		//for _, binding := range bindingNodes {
		//	bindingName := binding.SelectAttr("name")
		//	// Find all operation nodes within this binding
		//	operationNodes := xmlquery.Find(binding, "./wsdl:operation|./operation")
		//	for _, node := range operationNodes {
		//		if err := p.parseOperation(node, "", bindingName); err != nil {
		//			return err
		//		}
		//	}
		//}
	}

	return nil
}

func (p *wsdl1Parser) parseOperation(opNode *xmlquery.Node, portTypeName string) error {
	opName := opNode.SelectAttr("name")
	op := &Operation{
		Name: opName,
	}

	// Find corresponding binding operation to get SOAPAction, binding name and message parts (optional)
	bindingOp := p.findBindingOperation(portTypeName, opName)
	if bindingOp != nil {
		if soapActionNode := xmlquery.FindOne(bindingOp, "./soap:operation|./soap12:operation"); soapActionNode != nil {
			op.SOAPAction = soapActionNode.SelectAttr("soapAction")
		}
		// Get binding name from parent binding node
		if bindingNode := xmlquery.FindOne(bindingOp, "ancestor::wsdl:binding|ancestor::binding"); bindingNode != nil {
			op.Binding = bindingNode.SelectAttr("name")
		}
	}

	// Parse input message
	if inputNode := xmlquery.FindOne(opNode, "./wsdl:input|./input"); inputNode != nil {
		if msg, err := p.getMessage(inputNode, "input", bindingOp); err != nil {
			return fmt.Errorf("failed to parse input message: %w", err)
		} else if msg != nil {
			op.Input = msg
		}
	}

	// Parse output message
	if outputNode := xmlquery.FindOne(opNode, "./wsdl:output|./output"); outputNode != nil {
		if msg, err := p.getMessage(outputNode, "output", bindingOp); err != nil {
			return fmt.Errorf("failed to parse output message: %w", err)
		} else if msg != nil {
			op.Output = msg
		}
	}

	// Parse fault message
	if faultNode := xmlquery.FindOne(opNode, "./wsdl:fault|./fault"); faultNode != nil {
		if msg, err := p.getMessage(faultNode, "fault", bindingOp); err != nil {
			return fmt.Errorf("failed to parse fault message: %w", err)
		} else if msg != nil {
			op.Fault = msg
		}
	}

	p.operations[op.Name] = op
	return nil
}

// findBindingOperation finds the binding operation node for a given portType and operation name
func (p *wsdl1Parser) findBindingOperation(portTypeName, opName string) *xmlquery.Node {
	// First find the binding for this portType
	// Try with and without tns: prefix, and with both wsdl: and no prefix
	bindingExpr := fmt.Sprintf(`//wsdl:binding[@type='tns:%[1]s']|//binding[@type='tns:%[1]s']|//wsdl:binding[@type='%[1]s']|//binding[@type='%[1]s']`, portTypeName)
	bindingNode := xmlquery.FindOne(p.doc, bindingExpr)
	if bindingNode == nil {
		return nil
	}

	// Then find the operation within this binding
	return xmlquery.FindOne(bindingNode, fmt.Sprintf("./wsdl:operation[@name='%s']|./operation[@name='%s']", opName, opName))
}

// getMessage extracts message details from a WSDL message reference
func (p *wsdl1Parser) getMessage(msgNode *xmlquery.Node, messageType string, bindingOp *xmlquery.Node) (*wsdlmsg.Message, error) {
	// Get the message QName (e.g. "tns:GetPetByNameRequest")
	msgQName := msgNode.SelectAttr("message")
	if msgQName == "" {
		return nil, fmt.Errorf("no message attribute found for node: %s", msgNode.Data)
	}

	// Extract local part and prefix of the QName
	localPart := getLocalPart(msgQName)
	prefix := getPrefix(msgQName)

	// Look up the message definition, trying both with and without namespace prefix
	msgDef := xmlquery.FindOne(p.doc, fmt.Sprintf("/wsdl:definitions/wsdl:message[@name='%s']|/definitions/message[@name='%s']", localPart, localPart))
	if msgDef == nil && prefix != "" {
		// Try with the full QName
		msgDef = xmlquery.FindOne(p.doc, fmt.Sprintf("/wsdl:definitions/wsdl:message[@name='%s']|/definitions/message[@name='%s']", msgQName, msgQName))
	}
	if msgDef == nil {
		return nil, fmt.Errorf("message definition not found: %s", msgQName)
	}

	// Get the message parts
	partNodes := xmlquery.Find(msgDef, "./wsdl:part|./part")
	if len(partNodes) == 0 {
		return nil, fmt.Errorf("no parts found in message: %s", msgQName)
	}

	// Filter parts based on soap:body parts attribute
	partNodes = p.filterParts(bindingOp, messageType, partNodes)

	parts, err := p.parseParts(partNodes, msgQName)
	if err != nil {
		return nil, err
	}
	switch len(parts) {
	case 0:
		return nil, fmt.Errorf("message part must have either element or type attribute: %s", msgQName)
	case 1:
		return &parts[0], nil
	default:
		var composite wsdlmsg.Message = &wsdlmsg.CompositeMessage{Parts: &parts, MessageName: localPart}
		return &composite, nil
	}
}

// filterParts filters message parts based on the soap:body parts attribute
func (p *wsdl1Parser) filterParts(
	bindingOp *xmlquery.Node,
	messageType string,
	partNodes []*xmlquery.Node,
) []*xmlquery.Node {
	var partFilter *[]string

	// TODO use namespace-aware XPath to query for ./wsdl:input/soap:body (or ./input/soap:body)
	bindingOpSoapBodyNode := xmlquery.FindOne(bindingOp, fmt.Sprintf("./wsdl:%[1]s/soap:body|./wsdl:%[1]s/soap12:body", messageType))
	if bindingOpSoapBodyNode == nil {
		logger.Warnf("no soap:body found in binding operation: %s", bindingOp.Data)
	} else {
		msgParts := strings.Split(bindingOpSoapBodyNode.SelectAttr("parts"), " ")
		msgParts = utils.RemoveEmptyStrings(msgParts)
		if len(msgParts) > 0 {
			partFilter = &msgParts
		}
	}

	if partFilter != nil {
		var filteredParts []*xmlquery.Node
		for _, partNode := range partNodes {
			partNodeName := partNode.SelectAttr("name")
			if utils.StringSliceContainsElement(partFilter, partNodeName) {
				filteredParts = append(filteredParts, partNode)
			}
		}
		logger.Tracef("filtered parts: %v", filteredParts)
		partNodes = filteredParts
	}
	return partNodes
}

// parseParts parses message parts from a list of part nodes
func (p *wsdl1Parser) parseParts(
	partNodes []*xmlquery.Node,
	msgQName string,
) ([]wsdlmsg.Message, error) {
	schemas := *p.schemas

	var parts []wsdlmsg.Message
	for _, part := range partNodes {
		partName := part.SelectAttr("name")

		if element := part.SelectAttr("element"); element != "" {
			element = p.toQName(element)
			elementNode, err := schemas.ResolveElement(element)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve element %s: %w", element, err)
			}
			part := &wsdlmsg.ElementMessage{Element: elementNode}
			parts = append(parts, part)

		} else if typeRef := part.SelectAttr("type"); typeRef != "" {
			typeRef = p.toQName(typeRef)
			typeNode, err := schemas.ResolveType(typeRef)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve type %s: %w", typeRef, err)
			}
			part := &wsdlmsg.TypeMessage{Type: typeNode, PartName: partName}
			parts = append(parts, part)

		} else {
			return nil, fmt.Errorf("message part must have either element or type attribute: %s", msgQName)
		}
	}
	return parts, nil
}
