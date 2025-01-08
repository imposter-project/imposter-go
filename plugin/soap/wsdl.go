package soap

import (
	"fmt"
	"os"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/config"
)

// WSDLVersion represents the version of WSDL being used
type WSDLVersion int

const (
	WSDL1 WSDLVersion = iota + 1
	WSDL2
)

// SOAPVersion represents the version of SOAP being used
type SOAPVersion int

const (
	SOAP11 SOAPVersion = iota + 1
	SOAP12
)

// WSDLParser is the interface that all WSDL parsers must implement
type WSDLParser interface {
	GetVersion() WSDLVersion
	GetSOAPVersion() SOAPVersion
	GetOperations() map[string]*Operation
	GetOperation(name string) *Operation
	ValidateRequest(operation string, body []byte) error
	GetBindingName(op *Operation) string
}

// Operation represents a WSDL operation
type Operation struct {
	Name       string
	SOAPAction string
	Input      *Message
	Output     *Message
	Fault      *Message
	Binding    string
}

// Message represents a WSDL message
type Message struct {
	Name    string
	Element string
}

// BaseWSDLParser provides common functionality for WSDL parsers
type BaseWSDLParser struct {
	doc        *xmlquery.Node
	wsdlPath   string
	operations map[string]*Operation
}

// NewWSDLParser creates a new version-aware WSDL parser instance
func NewWSDLParser(wsdlPath string) (WSDLParser, error) {
	// Read and parse the WSDL file
	file, err := os.Open(wsdlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open WSDL file: %w", err)
	}
	defer file.Close()

	doc, err := xmlquery.Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WSDL file: %w", err)
	}

	// Detect WSDL version from root element namespace
	root := doc.SelectElement("*")
	if root == nil {
		return nil, fmt.Errorf("invalid WSDL document: no root element")
	}

	// Check if root has namespace attribute
	if len(root.Attr) == 0 {
		return nil, fmt.Errorf("invalid WSDL document: root element has no namespace")
	}

	// Check for WSDL 2.0
	for _, attr := range root.Attr {
		if strings.Contains(attr.Value, "http://www.w3.org/ns/wsdl") {
			return newWSDL2Parser(doc, wsdlPath)
		}
	}

	// Check for WSDL 1.1
	// check all root attributes
	for _, attr := range root.Attr {
		if strings.Contains(attr.Value, "http://schemas.xmlsoap.org/wsdl/") {
			return newWSDL1Parser(doc, wsdlPath)
		}
	}

	return nil, fmt.Errorf("unsupported WSDL version")
}

// WSDL 1.1 Parser
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

func (p *wsdl1Parser) GetBindingName(op *Operation) string {
	if op == nil {
		return ""
	}
	return op.Binding
}

func (p *wsdl2Parser) GetBindingName(op *Operation) string {
	if op == nil {
		return ""
	}
	return op.Binding
}

// AugmentConfigWithWSDL enriches the configuration with auto-generated interceptors for each WSDL operation.
func AugmentConfigWithWSDL(cfg *config.Config, parser WSDLParser) error {
	ops := parser.GetOperations()
	for _, op := range ops {
		// Create an interceptor with default RequestMatcher
		newInterceptor := config.Interceptor{
			Continue: true,
			RequestMatcher: config.RequestMatcher{
				Method:     "POST",
				SOAPAction: op.SOAPAction,
				Operation:  op.Name,
				Binding:    parser.GetBindingName(op),
				Capture: map[string]config.Capture{
					"_matched-soap-operation": config.Capture{
						Store: "request",
						CaptureConfig: config.CaptureConfig{
							Const: "true",
						},
					},
				},
			},
			Response: &config.Response{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": "application/soap+xml",
				},
				// Minimal placeholder for a SOAP response
				Content: `<soap:Envelope xmlns:soap="` + getNamespace(parser.GetSOAPVersion()) + `">
  <soap:Body>
    <!-- Example response for ` + op.Name + ` -->
  </soap:Body>
</soap:Envelope>`,
				},
			}
		cfg.Interceptors = append(cfg.Interceptors, newInterceptor)
	}

	// Add a default resource to handle unmatched requests
	defaultResource := config.Resource{
		RequestMatcher: config.RequestMatcher{
			AllOf: []config.ExpressionMatchCondition{
				{
					Expression: "${stores.request._matched-soap-operation}",
					MatchCondition: config.MatchCondition{
						Operator: "EqualTo",
						Value: "true",
					},
				},
			},
		},
		Response: config.Response{},
	}
	cfg.Resources = append(cfg.Resources, defaultResource)

	return nil
}

// getNamespace returns SOAP envelope namespace for the specified SOAP version
func getNamespace(version SOAPVersion) string {
	if version == SOAP12 {
		return "http://www.w3.org/2003/05/soap-envelope"
	}
	return "http://schemas.xmlsoap.org/soap/envelope/"
}
