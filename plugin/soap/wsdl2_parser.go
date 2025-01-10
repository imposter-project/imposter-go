package soap

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
)

// WSDL 2.0 Parser
type wsdl2Parser struct {
	BaseWSDLParser
}

func newWSDL2Parser(doc *xmlquery.Node, wsdlPath string) (*wsdl2Parser, error) {
	parser := &wsdl2Parser{
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

func (p *wsdl2Parser) GetVersion() WSDLVersion {
	return WSDL2
}

func (p *wsdl2Parser) GetSOAPVersion() SOAPVersion {
	// WSDL 2.0 typically uses SOAP 1.2
	return SOAP12
}

func (p *wsdl2Parser) GetOperations() map[string]*Operation {
	return p.operations
}

func (p *wsdl2Parser) GetOperation(name string) *Operation {
	return p.operations[name]
}

func (p *wsdl2Parser) ValidateRequest(operation string, body []byte) error {
	// TODO: Implement schema validation
	return nil
}

func (p *wsdl2Parser) parseOperations() error {
	// Find all interface nodes
	interfaceNodes := xmlquery.Find(p.doc, "//wsdl:interface|//interface")
	for _, iface := range interfaceNodes {
		interfaceName := iface.SelectAttr("name")
		operationNodes := xmlquery.Find(iface, "./wsdl:operation|./operation")
		for _, node := range operationNodes {
			opName := node.SelectAttr("name")
			op := &Operation{
				Name: opName,
			}

			// Parse input message
			if msg, err := p.getMessage(node, "./wsdl:input|./input", true); err != nil {
				return fmt.Errorf("failed to parse input message: %w", err)
			} else if msg != nil {
				op.Input = msg
			}

			// Parse output message
			if msg, err := p.getMessage(node, "./wsdl:output|./output", true); err != nil {
				return fmt.Errorf("failed to parse output message: %w", err)
			} else if msg != nil {
				op.Output = msg
			}

			// Try fault at operation level first, then interface level
			if msg, err := p.getMessage(node, "./wsdl:fault|./fault|./wsdl:outfault|./outfault", false); err != nil {
				return fmt.Errorf("failed to parse fault message: %w", err)
			} else if msg != nil {
				op.Fault = msg
			} else {
				// Try interface level fault
				if msg, err := p.getMessage(iface, "./wsdl:fault|./fault", false); err != nil {
					return fmt.Errorf("failed to parse interface fault message: %w", err)
				} else if msg != nil {
					op.Fault = msg
				}
			}

			// Find corresponding binding operation to get SOAPAction
			bindingOp := p.findBindingOperation(interfaceName, opName)
			if bindingOp != nil {
				if soapActionNode := xmlquery.FindOne(bindingOp, "./wsoap:operation"); soapActionNode != nil {
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

// findBindingOperation finds the binding operation node for a given interface and operation name
func (p *wsdl2Parser) findBindingOperation(interfaceName, opName string) *xmlquery.Node {
	// First find the binding for this interface
	bindingExpr := fmt.Sprintf("//wsdl:binding[@interface='tns:%s']|//binding[@interface='tns:%s']|//wsdl:binding[@interface='%s']|//binding[@interface='%s']", interfaceName, interfaceName, interfaceName, interfaceName)
	bindingNode := xmlquery.FindOne(p.doc, bindingExpr)
	if bindingNode == nil {
		return nil
	}

	// Then find the operation within this binding
	return xmlquery.FindOne(bindingNode, fmt.Sprintf("./wsdl:operation[@ref='tns:%s']|./operation[@ref='tns:%s']|./wsdl:operation[@ref='%s']|./operation[@ref='%s']", opName, opName, opName, opName))
}

// getMessage extracts message details from a WSDL 2.0 message reference
func (p *wsdl2Parser) getMessage(context *xmlquery.Node, expression string, required bool) (*Message, error) {
	msgNode := xmlquery.FindOne(context, expression)
	if msgNode == nil {
		if required {
			return nil, fmt.Errorf("required message not found: %s", expression)
		}
		return nil, nil
	}

	// WSDL 2.0 only allows element references (not type references)
	element := msgNode.SelectAttr("element")
	if element == "" {
		if required {
			return nil, fmt.Errorf("element attribute is required for WSDL 2.0 messages")
		}
		return nil, nil
	}

	// If the element reference is not qualified and we have a target namespace, qualify it
	if !strings.Contains(element, ":") {
		tns := p.GetTargetNamespace()
		if tns != "" {
			element = "tns:" + element
		}
	}

	return &Message{
		Name:    msgNode.SelectAttr("messageLabel"),
		Element: element,
	}, nil
}
