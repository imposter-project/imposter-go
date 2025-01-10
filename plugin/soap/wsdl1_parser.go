package soap // WSDL 1.1 Parser

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
)

type wsdl1Parser struct {
	BaseWSDLParser
}

func newWSDL1Parser(doc *xmlquery.Node, wsdlPath string) (*wsdl1Parser, error) {
	parser := &wsdl1Parser{
		BaseWSDLParser: BaseWSDLParser{
			doc:        doc,
			wsdlPath:   wsdlPath,
			operations: make(map[string]*Operation),
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
	// Find all portType nodes first
	portTypeNodes := xmlquery.Find(p.doc, "//wsdl:portType|//portType")
	for _, portType := range portTypeNodes {
		portTypeName := portType.SelectAttr("name")
		// Find all operation nodes within this portType
		operationNodes := xmlquery.Find(portType, "./wsdl:operation|./operation")
		for _, node := range operationNodes {
			opName := node.SelectAttr("name")
			op := &Operation{
				Name: opName,
			}

			// Parse input message
			if inputNode := xmlquery.FindOne(node, "./wsdl:input|./input"); inputNode != nil {
				if msg, err := p.getMessage(inputNode); err != nil {
					return fmt.Errorf("failed to parse input message: %w", err)
				} else {
					op.Input = msg
				}
			}

			// Parse output message
			if outputNode := xmlquery.FindOne(node, "./wsdl:output|./output"); outputNode != nil {
				if msg, err := p.getMessage(outputNode); err != nil {
					return fmt.Errorf("failed to parse output message: %w", err)
				} else {
					op.Output = msg
				}
			}

			// Parse fault message
			if faultNode := xmlquery.FindOne(node, "./wsdl:fault|./fault"); faultNode != nil {
				if msg, err := p.getMessage(faultNode); err != nil {
					return fmt.Errorf("failed to parse fault message: %w", err)
				} else {
					op.Fault = msg
				}
			}

			// Find corresponding binding operation to get SOAPAction and binding name
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

			p.operations[op.Name] = op
		}
	}

	return nil
}

// findBindingOperation finds the binding operation node for a given portType and operation name
func (p *wsdl1Parser) findBindingOperation(portTypeName, opName string) *xmlquery.Node {
	// First find the binding for this portType
	bindingExpr := fmt.Sprintf("//wsdl:binding[@type='tns:%s']|//binding[@type='tns:%s']|//wsdl:binding[@type='%s']|//binding[@type='%s']", portTypeName, portTypeName, portTypeName, portTypeName)
	bindingNode := xmlquery.FindOne(p.doc, bindingExpr)
	if bindingNode == nil {
		return nil
	}

	// Then find the operation within this binding
	return xmlquery.FindOne(bindingNode, fmt.Sprintf("./wsdl:operation[@name='%s']|./operation[@name='%s']", opName, opName))
}

// getMessage extracts message details from a WSDL message reference
func (p *wsdl1Parser) getMessage(msgNode *xmlquery.Node) (*Message, error) {
	// First try to get element attribute directly (for test cases)
	if element := msgNode.SelectAttr("element"); element != "" {
		return &Message{
			Name:    msgNode.SelectAttr("name"),
			Element: element,
		}, nil
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

	// For now, we'll use the first part's element or type
	part := parts[0]
	msg := &Message{
		Name: msgNode.SelectAttr("name"),
	}

	// Check for element reference
	if element := part.SelectAttr("element"); element != "" {
		// If the element reference is not qualified and we have a target namespace, qualify it
		if !strings.Contains(element, ":") {
			tns := p.GetTargetNamespace()
			if tns != "" {
				element = "tns:" + element
			}
		}
		msg.Element = element
		return msg, nil
	}

	// Check for type reference
	if typeRef := part.SelectAttr("type"); typeRef != "" {
		// If the type reference is not qualified and we have a target namespace, qualify it
		if !strings.Contains(typeRef, ":") {
			tns := p.GetTargetNamespace()
			if tns != "" {
				typeRef = "tns:" + typeRef
			}
		}
		msg.Type = typeRef
		msg.Name = part.SelectAttr("name") // For type references, use the part name
		return msg, nil
	}

	return nil, fmt.Errorf("message part must have either element or type attribute: %s", msgQName)
}
