package soap

import (
	"encoding/xml"
	"fmt"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/internal/wsdlmsg"
	"github.com/imposter-project/imposter-go/pkg/xsd"
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
	WSDL1Namespace   = "http://schemas.xmlsoap.org/wsdl/"
	WSDL2Namespace   = "http://www.w3.org/ns/wsdl"
	SOAP11Namespace  = "http://schemas.xmlsoap.org/wsdl/soap/"
	SOAP12Namespace  = "http://schemas.xmlsoap.org/wsdl/soap12/"
	WSOAP20Namespace = "http://www.w3.org/ns/wsdl/soap"
)

// WSDLDocProvider is the interface that provides the WSDL document
type WSDLDocProvider interface {
	GetWSDLDoc() *xmlquery.Node
	GetWSDLPath() string
	GetSchemaSystem() *xsd.SchemaSystem
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

// BaseWSDLParser provides common functionality for WSDL parsers
type BaseWSDLParser struct {
	wsdlPath        string
	doc             *xmlquery.Node
	operations      map[string]*Operation
	schemas         *xsd.SchemaSystem
	targetNamespace string
}

func (p *BaseWSDLParser) GetWSDLPath() string {
	return p.wsdlPath
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

// GetSchemaSystem returns the schema system
func (p *BaseWSDLParser) GetSchemaSystem() *xsd.SchemaSystem {
	return p.schemas
}

// GetTargetNamespace returns the target namespace of the WSDL document
func (p *BaseWSDLParser) GetTargetNamespace() string {
	return p.targetNamespace
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

// toQName qualifies a node if it is not already qualified, and a targetNamespace is present
func (p *BaseWSDLParser) toQName(node string) string {
	// TODO check if this should be the targetNamespace from the element's schema
	if !strings.Contains(node, ":") {
		tns := p.GetTargetNamespace()
		if tns != "" {
			node = "tns:" + node
		}
	}
	return node
}

// resolveMessagesToElements generates synthetic schemas for non-element messages
func (p *BaseWSDLParser) resolveMessagesToElements() error {
	schemaSystem := *p.GetSchemaSystem()
	targetNamespace := p.GetTargetNamespace()

	for _, op := range p.GetOperations() {
		if op.Input != nil {
			inputMsg, err := resolveMessage(*op.Input, "input", op.Name, targetNamespace, schemaSystem, p.wsdlPath)
			if err != nil {
				return err
			}
			op.Input = &inputMsg
		}

		if op.Output != nil {
			outputMsg, err := resolveMessage(*op.Output, "output", op.Name, targetNamespace, schemaSystem, p.wsdlPath)
			if err != nil {
				return err
			}
			op.Output = &outputMsg
		}

		if op.Fault != nil {
			faultMsg, err := resolveMessage(*op.Fault, "fault", op.Name, targetNamespace, schemaSystem, p.wsdlPath)
			if err != nil {
				return err
			}
			op.Fault = &faultMsg
		}
	}
	return nil
}

// resolveMessage generates a synthetic schema for a message that is not an element
func resolveMessage(rawMsg wsdlmsg.Message, msgRole string, opName string, targetNamespace string, schemaSystem xsd.SchemaSystem, wsdlPath string) (wsdlmsg.Message, error) {
	filename := fmt.Sprintf("synthetic-%s-%s.xsd", opName, msgRole)

	switch rawMsg.GetMessageType() {
	case wsdlmsg.ElementMessageType:
		return rawMsg, nil

	case wsdlmsg.TypeMessageType:
		msg := rawMsg.(*wsdlmsg.TypeMessage)
		schema, elementName := wsdlmsg.CreateSinglePartSchema(msg.PartName, msg.Type, targetNamespace)

		if err := schemaSystem.ImportSchema(wsdlPath, filename, schema); err != nil {
			return nil, err
		}
		processedMsg := &wsdlmsg.ElementMessage{
			Element: &xml.Name{
				Local: getLocalPart(elementName),
				Space: targetNamespace,
			},
		}
		return processedMsg, nil

	case wsdlmsg.CompositeMessageType:
		msg := rawMsg.(*wsdlmsg.CompositeMessage)
		elementName := msg.MessageName
		imports := schemaSystem.GetSchemasWithTargetNamespace(targetNamespace)
		schema := wsdlmsg.CreateCompositePartSchema(elementName, *msg.Parts, targetNamespace, imports)

		if err := schemaSystem.ImportSchema(wsdlPath, filename, schema); err != nil {
			return nil, err
		}
		processedMsg := &wsdlmsg.ElementMessage{
			Element: &xml.Name{
				Local: getLocalPart(elementName),
				Space: targetNamespace,
			},
		}
		return processedMsg, nil
	}
	return nil, fmt.Errorf("unsupported message type: %T", rawMsg)
}

// newWSDLParser creates a new version-aware WSDL parser instance
func newWSDLParser(wsdlPath string) (WSDLParser, error) {
	logger.Tracef("loading WSDL file %s", wsdlPath)

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

	var parser WSDLParser

	// Check for WSDL 2.0
	for _, attr := range root.Attr {
		if strings.Contains(attr.Value, WSDL2Namespace) {
			if parser, err = newWSDL2Parser(doc, wsdlPath); err != nil {
				return nil, err
			}
			break
		}
	}

	if parser == nil {
		// Check for WSDL 1.1
		for _, attr := range root.Attr {
			if strings.Contains(attr.Value, WSDL1Namespace) {
				if parser, err = newWSDL1Parser(doc, wsdlPath); err != nil {
					return nil, err
				}
				break
			}
		}
	}

	if parser == nil {
		return nil, fmt.Errorf("unsupported WSDL version")
	}
	return parser, nil
}

// augmentConfigWithWSDL enriches the configuration with auto-generated interceptors for each WSDL operation.
func augmentConfigWithWSDL(cfg *config.Config, parser WSDLParser) error {
	ops := parser.GetOperations()
	for _, op := range ops {
		logger.Debugf("adding interceptor for operation %s with binding %s", op.Name, op.Binding)

		// Generate example response XML
		// TODO make this lazy; use a template placeholder function, such as ${soap.example('${op.Name}')}
		exampleXml, err := generateExampleXML(op.Output, parser.GetSchemaSystem())
		if err != nil {
			return err
		}
		exampleResponse := wrapInEnvelope(exampleXml, parser.GetSOAPVersion())

		// Create an interceptor with default RequestMatcher
		newInterceptor := config.Interceptor{
			Continue: true,
			RequestMatcher: config.RequestMatcher{
				Method:    "POST",
				Operation: op.Name,
				Binding:   parser.GetBindingName(op),
				Capture: map[string]config.Capture{
					"_matched-soap-operation": {
						Store: "request",
						CaptureConfig: config.CaptureConfig{
							Const: "true",
						},
					},
				},

				// SOAPAction header is not mandatory - don't be too strict if we match the operation and binding
				//SOAPAction: op.SOAPAction,
			},
			Response: &config.Response{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": getResponseContentType(parser.GetSOAPVersion()),
				},
				Content: exampleResponse,
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

// getEnvNamespace returns SOAP envelope namespace for the specified SOAP version
func getEnvNamespace(version SOAPVersion) string {
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
