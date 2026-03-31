package soap

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/pkg/logger"
	wsdlparser "github.com/outofcoffee/go-wsdl-parser"
)

// Type aliases for backward compatibility within this package
type WSDLVersion = wsdlparser.WSDLVersion
type WSDLParser = wsdlparser.WSDLParser
type Operation = wsdlparser.Operation

// Re-export WSDL version constants
var (
	WSDL1 = wsdlparser.WSDL1
	WSDL2 = wsdlparser.WSDL2
)

// SOAPVersion represents the version of SOAP being used
type SOAPVersion int

const (
	SOAP11 SOAPVersion = iota + 1
	SOAP12
)

const (
	SOAP11Namespace         = "http://schemas.xmlsoap.org/wsdl/soap/"
	SOAP12Namespace         = "http://schemas.xmlsoap.org/wsdl/soap12/"
	WSOAP20Namespace        = "http://www.w3.org/ns/wsdl/soap"
	SOAP11EnvNamespace      = "http://schemas.xmlsoap.org/soap/envelope/"
	SOAP12DraftEnvNamespace = "http://www.w3.org/2001/12/soap-envelope"
	SOAP12RecEnvNamespace   = "http://www.w3.org/2003/05/soap-envelope"
)

const (
	soapExamplePlaceholder = "${soap.example()}"
)

// augmentConfigWithWSDL enriches the configuration with auto-generated interceptors for each WSDL operation.
func augmentConfigWithWSDL(cfg *config.Config, parser WSDLParser) error {
	ops := parser.GetOperations()
	for _, op := range ops {
		logger.Debugf("adding interceptor for operation %s with binding %s", op.Name, op.Binding)

		// Create an interceptor with default RequestMatcher
		newInterceptor := config.Interceptor{
			Continue: true,
			BaseResource: config.BaseResource{
				RuntimeGenerated: true,
				RequestMatcher: config.RequestMatcher{
					Method:    "POST",
					Operation: op.Name,
					Binding:   parser.GetBindingName(op),

					// SOAPAction header is not mandatory - don't be too strict if we match the operation and binding
					//SOAPAction: op.SOAPAction,
				},
				Capture: map[string]config.Capture{
					"_matched-soap-operation": {
						Store: "request",
						CaptureConfig: config.CaptureConfig{
							Const: op.Name,
						},
					},
				},
				Response: &config.Response{
					StatusCode: 200,
					Content:    soapExamplePlaceholder,
				},
			},
		}
		cfg.Interceptors = append(cfg.Interceptors, newInterceptor)
	}

	// Add a default resource to handle unmatched requests
	defaultResource := config.Resource{
		BaseResource: config.BaseResource{
			RuntimeGenerated: true,
			RequestMatcher: config.RequestMatcher{
				AllOf: []config.ExpressionMatchCondition{
					{
						Expression: "${stores.request._matched-soap-operation}",
						MatchCondition: config.MatchCondition{
							Operator: "Exists",
						},
					},
				},
			},
			Response: &config.Response{},
		},
	}
	cfg.Resources = append(cfg.Resources, defaultResource)

	return nil
}
