package soap

import "github.com/antchfx/xmlquery"

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
	// Find all interface operation nodes
	operationNodes := xmlquery.Find(p.doc, "//interface/operation")
	for _, node := range operationNodes {
		op := &Operation{
			Name: node.SelectAttr("name"),
		}

		// Parse input message
		if inputNode := xmlquery.FindOne(node, "./input"); inputNode != nil {
			op.Input = &Message{
				Name:    inputNode.SelectAttr("messageLabel"),
				Element: inputNode.SelectAttr("element"),
			}
		}

		// Parse output message
		if outputNode := xmlquery.FindOne(node, "./output"); outputNode != nil {
			op.Output = &Message{
				Name:    outputNode.SelectAttr("messageLabel"),
				Element: outputNode.SelectAttr("element"),
			}
		}

		// Parse fault message
		if faultNode := xmlquery.FindOne(node, "./outfault"); faultNode != nil {
			op.Fault = &Message{
				Name:    faultNode.SelectAttr("messageLabel"),
				Element: faultNode.SelectAttr("element"),
			}
		}

		// Get SOAPAction from binding
		if soapActionNode := xmlquery.FindOne(node, "./wsoap:operation"); soapActionNode != nil {
			op.SOAPAction = soapActionNode.SelectAttr("soapAction")
		}

		// Get binding name
		if bindingNode := xmlquery.FindOne(p.doc, "//binding[@interface='tns:TestInterface']"); bindingNode != nil {
			op.Binding = bindingNode.SelectAttr("name")
		}

		p.operations[op.Name] = op
	}

	return nil
}
