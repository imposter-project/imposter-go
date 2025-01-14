package soap

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	commonInterceptor "github.com/imposter-project/imposter-go/internal/interceptor"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
)

// Constants for SOAP content types
const (
	textXMLContentType = "text/xml"
	soap11ContentType  = textXMLContentType
	soap12ContentType  = "application/soap+xml"
)

// getSoapAction extracts the SOAPAction from headers
func (h *PluginHandler) getSoapAction(r *http.Request) string {
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
func (h *PluginHandler) parseBody(body []byte) (*MessageBodyHolder, error) {
	doc, err := xmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	// Get the envelope element and its namespace
	envNode := xmlquery.FindOne(doc, "//*[local-name()='Envelope']")
	if envNode == nil {
		return nil, fmt.Errorf("no SOAP envelope found")
	}

	// Get the envelope namespace - check both xmlns and xmlns:prefix attributes
	var envNamespace string
	for _, attr := range envNode.Attr {
		if attr.Name.Space == "xmlns" || (attr.Name.Space == "" && attr.Name.Local == "xmlns") {
			envNamespace = attr.Value
			break
		}
	}

	// If no default namespace, check for prefixed namespace
	if envNamespace == "" {
		prefix := strings.Split(envNode.Data, ":")[0]
		for _, attr := range envNode.Attr {
			if attr.Name.Space == "xmlns" && attr.Name.Local == prefix {
				envNamespace = attr.Value
				break
			}
		}
	}

	// If still no namespace found, check parent elements
	if envNamespace == "" {
		parent := envNode.Parent
		for parent != nil {
			for _, attr := range parent.Attr {
				if attr.Name.Space == "xmlns" || (attr.Name.Space == "" && attr.Name.Local == "xmlns") {
					envNamespace = attr.Value
					break
				}
			}
			if envNamespace != "" {
				break
			}
			parent = parent.Parent
		}
	}

	// Extract the body element
	bodyNode := xmlquery.FindOne(doc, "//*[local-name()='Body']/*[1]")
	if bodyNode == nil {
		return nil, fmt.Errorf("no SOAP body element found")
	}

	// Validate SOAP version if namespace is found
	if envNamespace != "" {
		expectedNamespace := ""
		if h.wsdlParser.GetSOAPVersion() == SOAP12 {
			expectedNamespace = "http://www.w3.org/2003/05/soap-envelope"
		} else {
			expectedNamespace = "http://schemas.xmlsoap.org/soap/envelope/"
		}

		if envNamespace != expectedNamespace {
			return nil, fmt.Errorf("invalid SOAP version namespace: expected %s, got %s", expectedNamespace, envNamespace)
		}
	}

	return &MessageBodyHolder{
		BodyRootElement: bodyNode,
		EnvNamespace:    envNamespace,
	}, nil
}

// determineOperation determines the operation from SOAPAction and body
func (h *PluginHandler) determineOperation(soapAction string, bodyHolder *MessageBodyHolder) *Operation {
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

	namespace := bodyElement.NamespaceURI
	localName := bodyElement.Data

	// Match operations based on input message
	var matchedOps []*Operation
	for _, op := range h.wsdlParser.GetOperations() {
		if op.Input != nil {
			// Match by element
			if op.Input.Element != nil {
				// inputNS, inputName := splitQName(op.Input.Element)
				inputNS, inputName := op.Input.Element.Space, op.Input.Element.Local
				if inputNS == namespace && inputName == localName {
					matchedOps = append(matchedOps, op)
				}
			}

			// Match by type
			if op.Input.Type != nil {
				// TODO: implement type matching
				//matchedOps = append(matchedOps, op)
			}
		}
	}

	if len(matchedOps) == 1 {
		return matchedOps[0]
	}
	return nil
}

// calculateScore calculates how well a request matches a resource or interceptor
func (h *PluginHandler) calculateScore(reqMatcher *config.RequestMatcher, r *http.Request, body []byte, operation string, soapAction string, requestStore store.Store) (score int, isWildcard bool) {
	// Get system XML namespaces
	var systemNamespaces map[string]string
	if h.config.System != nil {
		systemNamespaces = h.config.System.XMLNamespaces
	}

	// Calculate base score using common request matcher fields
	baseScore, baseWildcard := matcher.CalculateMatchScore(reqMatcher, r, body, systemNamespaces, h.imposterConfig, requestStore)
	score = baseScore
	isWildcard = baseWildcard

	// Check SOAP-specific matches
	if reqMatcher.Operation != "" {
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

	return score, isWildcard
}

// HandleRequest processes incoming SOAP requests
func (h *PluginHandler) HandleRequest(r *http.Request, requestStore store.Store, responseState *response.ResponseState) {
	// Only handle POST requests for SOAP
	if r.Method != http.MethodPost {
		return // Let the main handler deal with non-POST requests
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
	for _, interceptorCfg := range h.config.Interceptors {
		score, _ := h.calculateScore(&interceptorCfg.RequestMatcher, r, body, op.Name, soapAction, requestStore)
		if score > 0 {
			logger.Infof("matched interceptor - method:%s, path:%s", r.Method, r.URL.Path)

			if !commonInterceptor.ProcessInterceptor(responseState, r, body, interceptorCfg, requestStore, h.imposterConfig, h.configDir, h) {
				responseState.Handled = true
				return // Short-circuit if interceptor continue is false
			}
		}
	}

	// Find matching resources
	var matches []matcher.MatchResult
	for _, resource := range h.config.Resources {
		score, isWildcard := h.calculateScore(&resource.RequestMatcher, r, body, op.Name, soapAction, requestStore)
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
		logger.Warnf("multiple equally specific matches, using the first")
	}

	// Capture request data
	capture.CaptureRequestData(h.imposterConfig, best.Resource.Capture, r, body, requestStore)

	// Process the response
	h.ProcessResponse(responseState, r, best.Resource.Response, requestStore)
	responseState.Handled = true
}

// sendSOAPFault sends a SOAP fault response
func (h *PluginHandler) sendSOAPFault(rs *response.ResponseState, message string, statusCode int) {
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

// ProcessResponse processes and sends the SOAP response
func (h *PluginHandler) ProcessResponse(rs *response.ResponseState, r *http.Request, resp config.Response, requestStore store.Store) {
	// Set content type for SOAP response
	rs.Headers["Content-Type"] = "application/soap+xml"

	// Process the response using common handler
	response.ProcessResponse(rs, r, resp, h.configDir, requestStore, h.imposterConfig)
}
