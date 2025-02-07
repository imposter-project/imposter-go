package soap

import (
	"fmt"
	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/pkg/wsdlmsg"
	"github.com/imposter-project/imposter-go/pkg/xsd"
)

// WSDL 2.0 Parser
type wsdl2Parser struct {
	BaseWSDLParser
}

func newWSDL2Parser(doc *xmlquery.Node, wsdlPath string) (*wsdl2Parser, error) {
	schemas, err := xsd.ExtractSchemas(wsdlPath, doc)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schemas: %w", err)
	}
	targetNamespace := xsd.GetTargetNamespace(doc)
	parser := &wsdl2Parser{
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

func (p *wsdl2Parser) GetVersion() WSDLVersion {
	return WSDL2
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
	// Find all bindings
	bindingNodes := xmlquery.Find(p.doc, "//wsdl:binding|//binding")
	for _, wsdlBinding := range bindingNodes {
		bindingName := wsdlBinding.SelectAttr("name")

		interfaceName := wsdlBinding.SelectAttr("interface")
		if interfaceName == "" {
			return fmt.Errorf("interface attribute is required for WSDL 2.0 bindings")
		}

		_, ifaceLocalName := xsd.SplitQName(interfaceName)

		// Find the interface node
		interfaceNode := xmlquery.FindOne(p.doc, fmt.Sprintf("//wsdl:interface[@name='tns:%[1]s']|//interface[@name='tns:%[1]s']|//wsdl:interface[@name='%[1]s']|//interface[@name='%[1]s']", ifaceLocalName))
		if interfaceNode == nil {
			return fmt.Errorf("interface not found for binding: %s", interfaceName)
		}

		// Find all operation nodes
		operationNodes := xmlquery.Find(wsdlBinding, "./wsdl:operation|./operation")
		for _, bindingOperation := range operationNodes {
			if err := p.parseOperation(bindingOperation, interfaceNode, bindingName); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *wsdl2Parser) parseOperation(bindingOperation *xmlquery.Node, interfaceNode *xmlquery.Node, bindingName string) error {
	opRef := bindingOperation.SelectAttr("ref")

	_, opRefLocalName := xsd.SplitQName(opRef)

	// get interface operation node
	interfaceOperation := xmlquery.FindOne(interfaceNode, fmt.Sprintf("./wsdl:operation[@name='%[1]s']|./operation[@name='%[1]s']", opRefLocalName))
	if interfaceOperation == nil {
		return fmt.Errorf("operation not found for binding: %s", opRefLocalName)
	}

	op := &Operation{
		Name:    interfaceOperation.SelectAttr("name"),
		Binding: bindingName,
	}

	// Parse input message
	if msg, err := p.getMessage(interfaceOperation, "./wsdl:input|./input", true); err != nil {
		return fmt.Errorf("failed to parse input message: %w", err)
	} else if msg != nil {
		op.Input = msg
	}

	// Parse output message
	if msg, err := p.getMessage(interfaceOperation, "./wsdl:output|./output", true); err != nil {
		return fmt.Errorf("failed to parse output message: %w", err)
	} else if msg != nil {
		op.Output = msg
	}

	// Try fault at operation level first, then interface level
	if msg, err := p.getMessage(interfaceOperation, "./wsdl:fault|./fault|./wsdl:outfault|./outfault", false); err != nil {
		return fmt.Errorf("failed to parse fault message: %w", err)
	} else if msg != nil {
		op.Fault = msg
	} else {
		// Try interface level fault
		if msg, err := p.getMessage(interfaceNode, "./wsdl:fault|./fault", false); err != nil {
			return fmt.Errorf("failed to parse interface fault message: %w", err)
		} else if msg != nil {
			op.Fault = msg
		}
	}

	soapOp := xmlquery.FindOne(interfaceOperation, "./wsoap:operation")
	if soapOp != nil {
		op.SOAPAction = soapOp.SelectAttr("soapAction")
	}

	p.operations[op.Name] = op
	return nil
}

// findBindingOperation finds the binding operation node for a given interface and operation name
func (p *wsdl2Parser) findBindingOperation(interfaceName, opName string) *xmlquery.Node {
	// First find the binding for this interface
	// Try with and without tns: prefix, and with both wsdl: and no prefix
	bindingExpr := fmt.Sprintf(`//wsdl:binding[@interface='tns:%[1]s']|//binding[@interface='tns:%[1]s']|//wsdl:binding[@interface='%[1]s']|//binding[@interface='%[1]s']`, interfaceName)
	bindingNode := xmlquery.FindOne(p.doc, bindingExpr)
	if bindingNode == nil {
		return nil
	}

	// Then find the operation within this binding
	// Try with and without tns: prefix
	return xmlquery.FindOne(bindingNode, fmt.Sprintf(`./wsdl:operation[@ref='tns:%[1]s']|./operation[@ref='tns:%[1]s']|./wsdl:operation[@ref='%[1]s']|./operation[@ref='%[1]s']`, opName))
}

// getMessage extracts message details from a WSDL 2.0 message reference
func (p *wsdl2Parser) getMessage(context *xmlquery.Node, expression string, required bool) (*wsdlmsg.Message, error) {
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

	// qualify the element name if necessary
	element = p.toQName(element)

	elementNode, err := (*p.schemas).ResolveElement(element)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve element %s: %w", element, err)
	}

	var message wsdlmsg.Message = &wsdlmsg.ElementMessage{Element: elementNode}
	return &message, nil
}
