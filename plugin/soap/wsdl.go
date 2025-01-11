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

const (
	WSDL1Namespace     = "http://schemas.xmlsoap.org/wsdl/"
	WSDL2Namespace     = "http://www.w3.org/ns/wsdl"
	XMLSchemaNamespace = "http://www.w3.org/2001/XMLSchema"
	SOAP11Namespace    = "http://schemas.xmlsoap.org/wsdl/soap/"
	SOAP12Namespace    = "http://schemas.xmlsoap.org/wsdl/soap12/"
	WSOAP20Namespace   = "http://www.w3.org/ns/wsdl/soap"
)

// WSDLDocProvider is the interface that provides the WSDL document
type WSDLDocProvider interface {
	GetWSDLDoc() *xmlquery.Node
	GetWSDLPath() string
}

// WSDLParser is the interface that all WSDL parsers must implement
type WSDLParser interface {
	WSDLDocProvider
	GetVersion() WSDLVersion
	GetSOAPVersion() SOAPVersion
	GetOperations() map[string]*Operation
	GetOperation(name string) *Operation
	ValidateRequest(operation string, body []byte) error
	GetBindingName(op *Operation) string
	GetTargetNamespace() string
}

func (w *BaseWSDLParser) GetWSDLPath() string {
	return w.wsdlPath
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
	Type    string // For WSDL 1.1 message parts that use type references
}

// BaseWSDLParser provides common functionality for WSDL parsers
type BaseWSDLParser struct {
	wsdlPath   string
	doc        *xmlquery.Node
	operations map[string]*Operation
}

// GetBindingName returns the binding name for the given operation
func (p *BaseWSDLParser) GetBindingName(op *Operation) string {
	if op == nil {
		return ""
	}
	return op.Binding
}

// GetOperation returns the operation by name
func (p *BaseWSDLParser) GetOperation(name string) *Operation {
	return p.operations[name]
}

// GetOperations returns all operations
func (p *BaseWSDLParser) GetOperations() map[string]*Operation {
	return p.operations
}

// GetWSDLDoc returns the WSDL document
func (p *BaseWSDLParser) GetWSDLDoc() *xmlquery.Node {
	return p.doc
}

// GetTargetNamespace returns the target namespace of the WSDL document
func (p *BaseWSDLParser) GetTargetNamespace() string {
	root := p.doc.SelectElement("*")
	if root == nil {
		return ""
	}
	for _, attr := range root.Attr {
		if attr.Name.Local == "targetNamespace" {
			return attr.Value
		}
	}
	return ""
}

// GetNamespaceByPrefix returns the namespace URI for a given prefix
func (p *BaseWSDLParser) GetNamespaceByPrefix(prefix string) string {
	root := p.doc.SelectElement("*")
	if root == nil {
		return ""
	}
	for _, attr := range root.Attr {
		if attr.Name.Space == "xmlns" && attr.Name.Local == prefix {
			return attr.Value
		}
	}
	return ""
}

// newWSDLParser creates a new version-aware WSDL parser instance
func newWSDLParser(wsdlPath string) (WSDLParser, error) {
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
		if strings.Contains(attr.Value, WSDL2Namespace) {
			return newWSDL2Parser(doc, wsdlPath)
		}
	}

	// Check for WSDL 1.1
	for _, attr := range root.Attr {
		if strings.Contains(attr.Value, WSDL1Namespace) {
			return newWSDL1Parser(doc, wsdlPath)
		}
	}

	return nil, fmt.Errorf("unsupported WSDL version")
}

// augmentConfigWithWSDL enriches the configuration with auto-generated interceptors for each WSDL operation.
func augmentConfigWithWSDL(cfg *config.Config, parser WSDLParser) error {
	ops := parser.GetOperations()
	for _, op := range ops {
		// Generate example response XML
		// TODO make this lazy; use a template placeholder function, such as ${soap.example('${op.Name}')}
		exampleResponse, err := generateExampleXML(op.Output.Element, parser.GetWSDLPath(), parser.GetWSDLDoc())
		if err != nil {
			return err
		}

		// Create an interceptor with default RequestMatcher
		newInterceptor := config.Interceptor{
			Continue: true,
			RequestMatcher: config.RequestMatcher{
				Method:     "POST",
				SOAPAction: op.SOAPAction,
				Operation:  op.Name,
				Binding:    parser.GetBindingName(op),
				Capture: map[string]config.Capture{
					"_matched-soap-operation": {
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
				// Use generated example response
				Content: `<soap:Envelope xmlns:soap="` + getNamespace(parser.GetSOAPVersion()) + `">
  <soap:Body>` + exampleResponse + `</soap:Body>
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
						Value:    "true",
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

// getLocalPart extracts the local part from a QName
func getLocalPart(qname string) string {
	if idx := strings.Index(qname, ":"); idx != -1 {
		return qname[idx+1:]
	}
	return qname
}

// getPrefix extracts the prefix from a QName
func getPrefix(qname string) string {
	if idx := strings.Index(qname, ":"); idx != -1 {
		return qname[:idx]
	}
	return ""
}

// resolveNamespace resolves a namespace prefix to its URI
func resolveNamespace(node *xmlquery.Node, prefix string) string {
	if prefix == "" {
		return ""
	}
	for _, attr := range node.Attr {
		if attr.Name.Space == "xmlns" && attr.Name.Local == prefix {
			return attr.Value
		}
	}
	if node.Parent != nil {
		return resolveNamespace(node.Parent, prefix)
	}
	return ""
}
