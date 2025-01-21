package soap // WSDL 1.1 Parser

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/logger"
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
	parser := &wsdl1Parser{
		BaseWSDLParser: BaseWSDLParser{
			doc:        doc,
			wsdlPath:   wsdlPath,
			operations: make(map[string]*Operation),
			schemas:    &schemas,
		},
	}
	if err := parser.parseOperations(); err != nil {
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
func (p *wsdl1Parser) getMessage(msgNode *xmlquery.Node, messageType string, bindingOp *xmlquery.Node) (*Message, error) {
	var partFilter *[]string
	bindingOpSoapBodyNode := xmlquery.FindOne(bindingOp, fmt.Sprintf("./wsdl:%[1]s/soap:body|./wsdl:%[1]s/soap12:body", messageType))
	if bindingOpSoapBodyNode == nil {
		logger.Warnf("no soap:body found in binding operation: %s", bindingOp.Data)
	} else {
		msgParts := strings.Split(bindingOpSoapBodyNode.SelectAttr("parts"), " ")
		partFilter = &msgParts
	}

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
	parts := xmlquery.Find(msgDef, "./wsdl:part|./part")
	if len(parts) == 0 {
		return nil, fmt.Errorf("no parts found in message: %s", msgQName)
	}

	if partFilter != nil {
		var filteredParts []*xmlquery.Node
		for i := 0; i < len(parts); i++ {
			for _, partName := range *partFilter {
				if partName == parts[i].SelectAttr("name") {
					filteredParts = append(filteredParts, parts[i])
				}
			}
		}
		logger.Tracef("filtered parts: %v", filteredParts)
		parts = filteredParts
	}

	// For now, we'll use the first part's element or type
	part := parts[0]
	msg := &Message{}

	schemas := *p.schemas

	// Check for element reference
	if element := part.SelectAttr("element"); element != "" {
		// If the element reference is not qualified and we have a target namespace, qualify it
		// TODO check if this should be the targetNamespace from the element's schema
		if !strings.Contains(element, ":") {
			tns := p.GetTargetNamespace()
			if tns != "" {
				element = "tns:" + element
			}
		}

		elementNode, err := schemas.ResolveElement(element)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve element %s: %w", element, err)
		}
		msg.Element = elementNode

		return msg, nil
	}

	// Check for type reference
	if typeRef := part.SelectAttr("type"); typeRef != "" {
		// If the type reference is not qualified, and we have a target namespace, qualify it
		// TODO check if this should be the targetNamespace from the element's schema
		if !strings.Contains(typeRef, ":") {
			tns := p.GetTargetNamespace()
			if tns != "" {
				typeRef = "tns:" + typeRef
			}
		}

		typeNode, err := schemas.ResolveType(typeRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve type %s: %w", typeRef, err)
		}
		msg.Type = typeNode

		return msg, nil
	}

	return nil, fmt.Errorf("message part must have either element or type attribute: %s", msgQName)
}
