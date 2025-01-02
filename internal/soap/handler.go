package soap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/plugin"
	"github.com/imposter-project/imposter-go/internal/store"
)

// Constants for SOAP content types
const (
	textXMLContentType = "text/xml"
	soap11ContentType  = textXMLContentType
	soap12ContentType  = "application/soap+xml"
)

// SOAPEnvelope represents a SOAP envelope structure
type SOAPEnvelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    SOAPBody
}

// SOAP11Envelope represents a SOAP 1.1 envelope structure
type SOAP11Envelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    SOAP11Body
}

// SOAP12Envelope represents a SOAP 1.2 envelope structure
type SOAP12Envelope struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
	Body    SOAP12Body
}

// SOAPBody represents a SOAP body structure
type SOAPBody struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Content []byte   `xml:",innerxml"`
}

// SOAP11Body represents a SOAP 1.1 body structure
type SOAP11Body struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Content []byte   `xml:",innerxml"`
}

// SOAP12Body represents a SOAP 1.2 body structure
type SOAP12Body struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Body"`
	Content []byte   `xml:",innerxml"`
}

// SOAPFault represents a SOAP fault structure
type SOAPFault struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`
	Code    string   `xml:"faultcode"`
	String  string   `xml:"faultstring"`
	Detail  string   `xml:"detail,omitempty"`
}

// SOAP11Fault represents a SOAP 1.1 fault structure
type SOAP11Fault struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`
	Code    string   `xml:"faultcode"`
	String  string   `xml:"faultstring"`
	Detail  string   `xml:"detail,omitempty"`
}

// SOAP12Fault represents a SOAP 1.2 fault structure
type SOAP12Fault struct {
	XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Fault"`
	Code    struct {
		Value string `xml:"Value"`
	} `xml:"Code"`
	Reason struct {
		Text string `xml:"Text"`
	} `xml:"Reason"`
	Detail string `xml:"Detail,omitempty"`
}

// Handler handles SOAP requests based on WSDL configuration
type Handler struct {
	config     *config.Config
	configDir  string
	wsdlParser WSDLParser
}

// NewHandler creates a new SOAP handler
func NewHandler(cfg *config.Config, configDir string) (*Handler, error) {
	// If WSDLFile is not absolute, make it relative to configDir
	wsdlPath := cfg.WSDLFile
	if !filepath.IsAbs(wsdlPath) {
		wsdlPath = filepath.Join(configDir, wsdlPath)
	}

	parser, err := NewWSDLParser(wsdlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse WSDL: %w", err)
	}

	return &Handler{
		config:     cfg,
		configDir:  configDir,
		wsdlParser: parser,
	}, nil
}

// getSoapAction extracts the SOAPAction from headers
func (h *Handler) getSoapAction(r *http.Request) string {
	// Try SOAPAction header first
	if soapAction := r.Header.Get("SOAPAction"); soapAction != "" {
		return strings.Trim(soapAction, "\"")
	}

	// For SOAP 1.2, check Content-Type header for action parameter
	if h.wsdlParser.GetSOAPVersion() == SOAP12 {
		contentType := r.Header.Get("Content-Type")
		parts := strings.Split(contentType, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "action=") {
				return strings.Trim(strings.TrimPrefix(part, "action="), "\"")
			}
		}
	}

	return ""
}

// MessageBodyHolder represents a parsed SOAP message
type MessageBodyHolder struct {
	BodyRootElement *xmlquery.Node
	EnvNamespace    string
}

// parseBody parses the SOAP body based on configuration
func (h *Handler) parseBody(body []byte) (*MessageBodyHolder, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// For SOAP messages, extract the body element
	if h.wsdlParser.GetSOAPVersion() == SOAP12 {
		bodyNode := xmlquery.FindOne(doc, "//*[local-name()='Body']/*[1]")
		if bodyNode == nil {
			return nil, fmt.Errorf("no SOAP body element found")
		}
		return &MessageBodyHolder{
			BodyRootElement: bodyNode,
			EnvNamespace:    "http://www.w3.org/2003/05/soap-envelope",
		}, nil
	}

	// SOAP 1.1
	bodyNode := xmlquery.FindOne(doc, "//*[local-name()='Body']/*[1]")
	if bodyNode == nil {
		return nil, fmt.Errorf("no SOAP body element found")
	}
	return &MessageBodyHolder{
		BodyRootElement: bodyNode,
		EnvNamespace:    "http://schemas.xmlsoap.org/soap/envelope/",
	}, nil
}

// determineOperation determines the operation from SOAPAction and body
func (h *Handler) determineOperation(soapAction string, bodyHolder *MessageBodyHolder) *Operation {
	// Try matching by SOAPAction first
	if soapAction != "" {
		for _, op := range h.wsdlParser.GetOperations() {
			if op.SOAPAction == soapAction {
				return op
			}
		}
	}

	// Try matching by body element
	bodyElement := bodyHolder.BodyRootElement
	if bodyElement == nil {
		return nil
	}

	// Get namespace and local name
	var namespace string
	for _, attr := range bodyElement.Attr {
		if attr.Name.Space == "xmlns" {
			namespace = attr.Value
			break
		}
	}
	localName := bodyElement.Data

	// Remove Request suffix if present
	if strings.HasSuffix(localName, "Request") {
		localName = strings.TrimSuffix(localName, "Request")
	}

	// Match operations based on input message
	var matchedOps []*Operation
	for _, op := range h.wsdlParser.GetOperations() {
		if op.Input != nil {
			// Match by element
			if op.Input.Element != "" {
				inputNS, inputName := splitQName(op.Input.Element)
				if inputNS == namespace && inputName == localName {
					matchedOps = append(matchedOps, op)
				}
			}

			// Match by name
			if op.Input.Name == localName {
				matchedOps = append(matchedOps, op)
			}
		}
	}

	if len(matchedOps) == 1 {
		return matchedOps[0]
	}
	return nil
}

// splitQName splits a qualified name into namespace and local part
func splitQName(qname string) (namespace, localPart string) {
	parts := strings.Split(qname, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", qname
}

// matchesSOAPOperation checks if a resource matches the SOAP operation and action
func (h *Handler) matchesSOAPOperation(resource *config.Resource, operation, soapAction string) bool {
	// Check operation match if specified
	if resource.Operation != nil {
		if resource.Operation.Name != "" {
			// Remove 'Request' suffix from operation name if present
			if strings.HasSuffix(operation, "Request") {
				operation = strings.TrimSuffix(operation, "Request")
			}
			if resource.Operation.Name != operation {
				return false
			}
		}
		if resource.Operation.SOAPAction != "" && resource.Operation.SOAPAction != soapAction {
			return false
		}
	}

	// Check direct SOAPAction match if specified
	if resource.SOAPAction != "" && resource.SOAPAction != soapAction {
		return false
	}

	return true
}

// HandleRequest processes incoming SOAP requests
func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// Only handle POST requests for SOAP
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and parse the SOAP request
	body, err := plugin.GetRequestBody(r)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Initialize request-scoped store and response state
	requestStore := make(store.Store)
	responseState := plugin.NewResponseState()

	// Process interceptors first
	for _, interceptor := range h.config.Interceptors {
		score, isWildcard := plugin.CalculateMatchScore(&interceptor.RequestMatcher, r, body)
		if score > 0 {
			fmt.Printf("Matched interceptor - method:%s, path:%s, wildcard:%v\n",
				r.Method, r.URL.Path, isWildcard)

			// Capture request data if specified
			if interceptor.Capture != nil {
				capture.CaptureRequestData(nil, config.Resource{
					RequestMatcher: config.RequestMatcher{
						Capture: interceptor.Capture,
					},
				}, r, body, requestStore)
			}

			// If the interceptor has a response and continue is false, send the response and stop processing
			if interceptor.Response != nil {
				h.processResponse(responseState, r, *interceptor.Response, requestStore)
				if !interceptor.Continue {
					responseState.WriteToResponseWriter(w)
					return
				}
			}
		}
	}

	// Parse the SOAP body
	bodyHolder, err := h.parseBody(body)
	if err != nil {
		h.sendSOAPFault(responseState, "Invalid SOAP envelope", http.StatusBadRequest)
		responseState.WriteToResponseWriter(w)
		return
	}

	// Get SOAPAction from headers
	soapAction := h.getSoapAction(r)

	// Determine operation
	op := h.determineOperation(soapAction, bodyHolder)
	if op == nil {
		h.sendSOAPFault(responseState, "No matching SOAP operation found", http.StatusNotFound)
		responseState.WriteToResponseWriter(w)
		return
	}

	// Find matching resources
	var matches []plugin.MatchResult
	for _, resource := range h.config.Resources {
		score, isWildcard := plugin.CalculateMatchScore(&resource.RequestMatcher, r, body)
		if score > 0 && h.matchesSOAPOperation(&resource, op.Name, soapAction) {
			matches = append(matches, plugin.MatchResult{Resource: &resource, Score: score, Wildcard: isWildcard})
		}
	}

	if len(matches) == 0 {
		h.sendSOAPFault(responseState, "No matching SOAP operation found", http.StatusNotFound)
		responseState.WriteToResponseWriter(w)
		return
	}

	// Find the best match
	best, tie := plugin.FindBestMatch(matches)
	if tie {
		fmt.Printf("Warning: multiple equally specific matches. Using the first.\n")
	}

	// Capture request data
	capture.CaptureRequestData(nil, *best.Resource, r, body, requestStore)

	// Process the response
	h.processResponse(responseState, r, best.Resource.Response, requestStore)
	responseState.WriteToResponseWriter(w)
}

// sendSOAPFault sends a SOAP fault response
func (h *Handler) sendSOAPFault(rs *plugin.ResponseState, message string, statusCode int) {
	rs.Headers["Content-Type"] = "application/soap+xml"
	rs.StatusCode = statusCode

	var faultXML string
	if h.wsdlParser.GetSOAPVersion() == SOAP12 {
		faultXML = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope">
    <env:Body>
        <env:Fault>
            <env:Code>
                <env:Value>env:Receiver</env:Value>
            </env:Code>
            <env:Reason>
                <env:Text>%s</env:Text>
            </env:Reason>
        </env:Fault>
    </env:Body>
</env:Envelope>`, message)
	} else {
		faultXML = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="http://schemas.xmlsoap.org/soap/envelope/">
    <env:Body>
        <env:Fault>
            <faultcode>env:Server</faultcode>
            <faultstring>%s</faultstring>
        </env:Fault>
    </env:Body>
</env:Envelope>`, message)
	}

	rs.Body = []byte(faultXML)
}

// processResponse processes and sends the SOAP response
func (h *Handler) processResponse(rs *plugin.ResponseState, r *http.Request, response config.Response, requestStore store.Store) {
	// Handle delay if specified
	plugin.SimulateDelay(response.Delay, r)

	// Set content type for SOAP response
	rs.Headers["Content-Type"] = "application/soap+xml"

	// Set custom headers if any
	for key, value := range response.Headers {
		rs.Headers[key] = value
	}

	// Set status code
	if response.StatusCode != 0 {
		rs.StatusCode = response.StatusCode
	} else {
		rs.StatusCode = http.StatusOK
	}

	// Handle failure simulation
	if response.Fail != "" {
		if plugin.SimulateFailure(rs, response.Fail, r) {
			return
		}
	}

	// Write response content
	if response.File != "" {
		filePath := filepath.Join(h.configDir, response.File)
		data, err := os.ReadFile(filePath)
		if err != nil {
			rs.StatusCode = http.StatusInternalServerError
			rs.Body = []byte("Failed to read file")
			return
		}
		rs.Body = data
	} else {
		rs.Body = []byte(response.Content)
	}

	fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
		r.Method, r.URL.Path, rs.StatusCode, len(rs.Body))
}
