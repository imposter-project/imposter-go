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

// augmentConfigWithWSDL enriches the configuration with auto-generated interceptors for each WSDL operation.
func augmentConfigWithWSDL(cfg *config.Config, parser WSDLParser) error {
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
