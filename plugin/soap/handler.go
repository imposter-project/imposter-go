package soap

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/imposter-project/imposter-go/pkg/wsdlmsg"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/steps"
	"github.com/imposter-project/imposter-go/internal/store"
)

// MessageBodyHolder represents a parsed SOAP message
type MessageBodyHolder struct {
	BodyRootElement *xmlquery.Node
	EnvNamespace    string
}

// GetSOAPVersion returns the SOAP version based on the envelope namespace
func (b *MessageBodyHolder) GetSOAPVersion() SOAPVersion {
	switch b.EnvNamespace {
	case SOAP11EnvNamespace:
		return SOAP11
	case SOAP12DraftEnvNamespace:
		return SOAP12
	case SOAP12RecEnvNamespace:
		return SOAP12
	default:
		panic(fmt.Errorf("root element is not a SOAP envelope - namespace is %s", b.EnvNamespace))
	}
}

// getSoapAction extracts the SOAPAction from headers
func (h *PluginHandler) getSoapAction(r *http.Request, body *MessageBodyHolder) string {
	// Try SOAPAction header first
	if soapAction := r.Header.Get("SOAPAction"); soapAction != "" {
		return strings.Trim(soapAction, "\"")
	}

	// For SOAP 1.2, check Content-Type header for action parameter
	if body.GetSOAPVersion() == SOAP12 {
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
		// if a SOAPAction is present, but no matching operation is found, return nil
		return nil
	}

	// Try matching by body element
	bodyElement := bodyHolder.BodyRootElement
	if bodyElement == nil {
		return nil
	}

	bodyRootElementNs := bodyElement.NamespaceURI
	bodyRootElementLocal := bodyElement.Data

	// Match operations based on input message
	var matchedOps []*Operation
	for _, op := range h.wsdlParser.GetOperations() {
		if op.Input != nil {
			inputMsg := *op.Input

			switch inputMsg.GetMessageType() {
			case wsdlmsg.ElementMessageType:
				// Match by element
				elementMsg := inputMsg.(*wsdlmsg.ElementMessage)
				inputNS, inputName := elementMsg.Element.Space, elementMsg.Element.Local
				if inputNS == bodyRootElementNs && inputName == bodyRootElementLocal {
					matchedOps = append(matchedOps, op)
				}
			case wsdlmsg.TypeMessageType:
				// Match by type
				// TODO consider matching on body child element names against part names
				if bodyRootElementLocal == op.Name {
					matchedOps = append(matchedOps, op)
				}
			case wsdlmsg.CompositeMessageType:
				if inputMsg.(*wsdlmsg.CompositeMessage).MessageName == bodyRootElementLocal {
					matchedOps = append(matchedOps, op)
				}
			}
		}
	}

	if len(matchedOps) == 1 {
		return matchedOps[0]
	}
	return nil
}

// calculateScore calculates how well a request matches a resource or interceptor
func (h *PluginHandler) calculateScore(reqMatcher *config.RequestMatcher, r *http.Request, body []byte, op *Operation, soapAction string, requestStore *store.Store) (score int, isWildcard bool) {
	// Get system XML namespaces
	var systemNamespaces map[string]string
	if h.config.System != nil {
		systemNamespaces = h.config.System.XMLNamespaces
	}

	// Calculate base score using common request matcher fields
	baseScore, baseWildcard := matcher.CalculateMatchScore(reqMatcher, r, body, systemNamespaces, h.imposterConfig, requestStore)
	if baseScore == matcher.NegativeMatchScore {
		return matcher.NegativeMatchScore, false
	}
	score = baseScore
	isWildcard = baseWildcard

	// Check SOAP-specific matches
	if reqMatcher.Operation != "" {
		if reqMatcher.Operation != op.Name {
			return matcher.NegativeMatchScore, false
		}
		score++
	}

	if reqMatcher.SOAPAction != "" {
		if reqMatcher.SOAPAction != soapAction {
			return matcher.NegativeMatchScore, false
		}
		score++
	}

	if reqMatcher.Binding != "" {
		bindingName := h.wsdlParser.GetBindingName(op)
		if reqMatcher.Binding != bindingName {
			return matcher.NegativeMatchScore, false
		}
		score++
	}

	return score, isWildcard
}

// HandleRequest processes incoming SOAP requests
func (h *PluginHandler) HandleRequest(r *http.Request, requestStore *store.Store, responseState *response.ResponseState, respProc response.Processor) {
	// Only handle POST requests for SOAP
	if r.Method != http.MethodPost {
		return // Let the main handler deal with non-POST requests
	}

	wsdlVersion := h.wsdlParser.GetVersion()

	// Read and parse the SOAP request
	body, err := matcher.GetRequestBody(r)
	if err != nil {
		logger.Warnf("failed to read request body: %v", err)
		soapVersion := guessSoapVersion(wsdlVersion)
		h.failWithSOAPFault(soapVersion, guessEnvNamespace(soapVersion), responseState, "Failed to read request body", http.StatusBadRequest)
		responseState.Handled = true
		return
	}

	// Create exchange once at the top
	exch := &exchange.Exchange{
		Request: &exchange.RequestContext{
			Request: r,
			Body:    body,
		},
		RequestStore: requestStore,
	}

	// Parse the SOAP body first since we need it for both interceptors and resources
	bodyHolder, err := h.parseBody(body)
	if err != nil {
		logger.Warnf("failed to parse SOAP body: %v", err)
		soapVersion := guessSoapVersion(wsdlVersion)
		h.failWithSOAPFault(soapVersion, guessEnvNamespace(soapVersion), responseState, "Invalid SOAP envelope", http.StatusBadRequest)
		responseState.Handled = true
		return
	}

	// Get SOAPAction from headers
	soapAction := h.getSoapAction(r, bodyHolder)

	// Determine operation
	op := h.determineOperation(soapAction, bodyHolder)
	if op == nil {
		return // Let the main handler deal with no operation match
	}

	// Process interceptors first
	for _, interceptorCfg := range h.config.Interceptors {
		score, _ := h.calculateScore(&interceptorCfg.RequestMatcher, r, body, op, soapAction, requestStore)
		if score > 0 {
			logger.Infof("matched interceptor - method:%s, path:%s", r.Method, r.URL.Path)
			if interceptorCfg.Capture != nil {
				capture.CaptureRequestData(h.imposterConfig, interceptorCfg.Capture, exch)
			}

			// Execute steps if present
			if len(interceptorCfg.Steps) > 0 {
				if err := steps.RunSteps(interceptorCfg.Steps, exch, h.imposterConfig, h.configDir, responseState); err != nil {
					logger.Errorf("failed to execute interceptor steps: %v", err)
					soapVersion := bodyHolder.GetSOAPVersion()
					h.failWithSOAPFault(soapVersion, bodyHolder.EnvNamespace, responseState, "Failed to execute steps", http.StatusInternalServerError)
					responseState.Handled = true
					return
				}
				if responseState.Handled {
					// Step(s) handled the request, so we don't need to process the response
					return
				}
			}

			if interceptorCfg.Response != nil {
				h.processResponse(bodyHolder, &interceptorCfg.RequestMatcher, responseState, r, interceptorCfg.Response, requestStore, op, respProc)
			}
			if !interceptorCfg.Continue {
				responseState.Handled = true
				return // Short-circuit if interceptor continue is false
			}
		}
	}

	// Find matching resources
	var matches []matcher.MatchResult
	for _, resource := range h.config.Resources {
		score, isWildcard := h.calculateScore(&resource.RequestMatcher, r, body, op, soapAction, requestStore)
		if score > 0 {
			matches = append(matches, matcher.MatchResult{Resource: &resource, Score: score, Wildcard: isWildcard, RuntimeGenerated: resource.RuntimeGenerated})
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
	capture.CaptureRequestData(h.imposterConfig, best.Resource.Capture, exch)

	// Execute steps if present
	if len(best.Resource.Steps) > 0 {
		if err := steps.RunSteps(best.Resource.Steps, exch, h.imposterConfig, h.configDir, responseState); err != nil {
			logger.Errorf("failed to execute resource steps: %v", err)
			soapVersion := bodyHolder.GetSOAPVersion()
			h.failWithSOAPFault(soapVersion, bodyHolder.EnvNamespace, responseState, "Failed to execute steps", http.StatusInternalServerError)
			responseState.Handled = true
			return
		}
		if responseState.Handled {
			// Step(s) handled the request, so we don't need to process the response
			return
		}
	}

	// Process the response
	h.processResponse(bodyHolder, &best.Resource.RequestMatcher, responseState, r, &best.Resource.Response, requestStore, op, respProc)
	responseState.Handled = true
}
