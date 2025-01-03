package soap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/imposter-project/imposter-go/internal/response"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/matcher"
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
	config         *config.Config
	configDir      string
	wsdlParser     WSDLParser
	imposterConfig *config.ImposterConfig
}

// NewHandler creates a new SOAP handler
func NewHandler(cfg *config.Config, configDir string, imposterConfig *config.ImposterConfig) (*Handler, error) {
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
		config:         cfg,
		configDir:      configDir,
		wsdlParser:     parser,
		imposterConfig: imposterConfig,
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

// calculateScore calculates a unified match score for SOAP requests
func (h *Handler) calculateScore(reqMatcher *config.RequestMatcher, r *http.Request, body []byte, operation string, soapAction string) (score int, isWildcard bool) {
	// Get base score from matcher
	baseScore, baseWildcard := matcher.CalculateMatchScore(reqMatcher, r, body)
	score = baseScore
	isWildcard = baseWildcard

	// Check SOAP-specific matches
	if reqMatcher.Operation != "" {
		// Remove 'Request' suffix from operation name if present
		if strings.HasSuffix(operation, "Request") {
			operation = strings.TrimSuffix(operation, "Request")
		}
		if reqMatcher.Operation != operation {
			return 0, false
		}
		score++
	}

	if reqMatcher.SOAPAction != "" {
		if reqMatcher.SOAPAction != soapAction {
			return 0, false
		}
		score++
	}

	if reqMatcher.Binding != "" {
		op := h.wsdlParser.GetOperation(operation)
		if op == nil {
			return 0, false
		}
		bindingName := h.wsdlParser.GetBindingName(op)
		if reqMatcher.Binding != bindingName {
			return 0, false
		}
		score++
	}

	// If no matchers were specified at all, return 0
	if score == 0 && reqMatcher.Method == "" && reqMatcher.Path == "" &&
		reqMatcher.Operation == "" && reqMatcher.SOAPAction == "" && reqMatcher.Binding == "" &&
		len(reqMatcher.Headers) == 0 && len(reqMatcher.QueryParams) == 0 && len(reqMatcher.FormParams) == 0 {
		return 0, false
	}

	return score, isWildcard
}

// HandleRequest processes incoming SOAP requests
func (h *Handler) HandleRequest(r *http.Request, requestStore store.Store, responseState *response.ResponseState) {
	// Only handle POST requests for SOAP
	if r.Method != http.MethodPost {
		responseState.StatusCode = http.StatusMethodNotAllowed
		responseState.Body = []byte("Method not allowed")
		responseState.Handled = true
		return
	}

	// Read and parse the SOAP request
	body, err := matcher.GetRequestBody(r)
	if err != nil {
		h.sendSOAPFault(responseState, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse the SOAP body first since we need it for both interceptors and resources
	bodyHolder, err := h.parseBody(body)
	if err != nil {
		h.sendSOAPFault(responseState, "Invalid SOAP envelope", http.StatusBadRequest)
		responseState.Handled = true
		return
	}

	// Get SOAPAction from headers
	soapAction := h.getSoapAction(r)

	// Determine operation
	op := h.determineOperation(soapAction, bodyHolder)
	if op == nil {
		return // Let the main handler deal with no operation match
	}

	// Process interceptors first
	for _, interceptor := range h.config.Interceptors {
		score, isWildcard := h.calculateScore(&interceptor.RequestMatcher, r, body, op.Name, soapAction)
		if score > 0 {
			fmt.Printf("Matched interceptor - method:%s, path:%s, wildcard:%v\n",
				r.Method, r.URL.Path, isWildcard)

			// Capture request data if specified
			if interceptor.Capture != nil {
				capture.CaptureRequestData(h.imposterConfig, config.Resource{
					RequestMatcher: config.RequestMatcher{
						Capture: interceptor.Capture,
					},
				}, r, body, requestStore)
			}

			// If the interceptor has a response and continue is false, send the response and stop processing
			if interceptor.Response != nil {
				h.processResponse(responseState, r, *interceptor.Response, requestStore)
				if !interceptor.Continue {
					responseState.Handled = true
					return
				}
			}
		}
	}

	// Find matching resources
	var matches []matcher.MatchResult
	for _, resource := range h.config.Resources {
		score, isWildcard := h.calculateScore(&resource.RequestMatcher, r, body, op.Name, soapAction)
		if score > 0 {
			matches = append(matches, matcher.MatchResult{Resource: &resource, Score: score, Wildcard: isWildcard})
		}
	}

	if len(matches) == 0 {
		return // Let the main handler deal with no matches
	}

	// Find the best match
	best, tie := matcher.FindBestMatch(matches)
	if tie {
		fmt.Printf("Warning: multiple equally specific matches. Using the first.\n")
	}

	// Capture request data
	capture.CaptureRequestData(h.imposterConfig, *best.Resource, r, body, requestStore)

	// Process the response
	h.processResponse(responseState, r, best.Resource.Response, requestStore)
	responseState.Handled = true
}

// sendSOAPFault sends a SOAP fault response
func (h *Handler) sendSOAPFault(rs *response.ResponseState, message string, statusCode int) {
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
func (h *Handler) processResponse(rs *response.ResponseState, r *http.Request, resp config.Response, requestStore store.Store) {
	// Set content type for SOAP response
	rs.Headers["Content-Type"] = "application/soap+xml"

	// Process the response using common handler
	response.ProcessResponse(rs, r, resp, h.configDir, requestStore, nil)
}