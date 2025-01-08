package soap // WSDL 1.1 Parser

import "github.com/antchfx/xmlquery"

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
	// Find all operation nodes
	operationNodes := xmlquery.Find(p.doc, "//wsdl:operation|//operation")
	for _, node := range operationNodes {
		op := &Operation{
			Name: node.SelectAttr("name"),
		}

		// Parse input message
		if inputNode := xmlquery.FindOne(node, "./wsdl:input|./input"); inputNode != nil {
			op.Input = &Message{
				Name:    inputNode.SelectAttr("name"),
				Element: inputNode.SelectAttr("element"),
			}
		}

		// Parse output message
		if outputNode := xmlquery.FindOne(node, "./wsdl:output|./output"); outputNode != nil {
			op.Output = &Message{
				Name:    outputNode.SelectAttr("name"),
				Element: outputNode.SelectAttr("element"),
			}
		}

		// Parse fault message
		if faultNode := xmlquery.FindOne(node, "./wsdl:fault|./fault"); faultNode != nil {
			op.Fault = &Message{
				Name:    faultNode.SelectAttr("name"),
				Element: faultNode.SelectAttr("element"),
			}
		}

		// Get SOAPAction from binding
		if soapActionNode := xmlquery.FindOne(node, "./soap:operation|./soap12:operation"); soapActionNode != nil {
			op.SOAPAction = soapActionNode.SelectAttr("soapAction")
		}

		// Get binding name
		if bindingNode := xmlquery.FindOne(node, "ancestor::wsdl:binding|ancestor::binding"); bindingNode != nil {
			op.Binding = bindingNode.SelectAttr("name")
		}

		p.operations[op.Name] = op
	}

	return nil
}
