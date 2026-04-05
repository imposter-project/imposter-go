package soap

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/outofcoffee/go-wsdl-parser/wsdlmsg"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/pipeline"
	"github.com/imposter-project/imposter-go/internal/response"
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
				// RPC-style: the SOAP body root element is the operation name
				// wrapper, and its children are the individual message parts.
				// Match by the wrapper element's local name against the WSDL
				// operation name, scoped by the WSDL target namespace when
				// the request supplies one.
				if bodyRootElementLocal == op.Name && namespaceMatchesTarget(bodyRootElementNs, h.wsdlParser.GetTargetNamespace()) {
					matchedOps = append(matchedOps, op)
				}
			case wsdlmsg.CompositeMessageType:
				// A composite input may come from an RPC-style operation
				// (wrapper named after the operation) or a document-style
				// message with multiple parts (wrapper named after the
				// message). Accept either.
				compositeMsg := inputMsg.(*wsdlmsg.CompositeMessage)
				if (compositeMsg.MessageName == bodyRootElementLocal || op.Name == bodyRootElementLocal) &&
					namespaceMatchesTarget(bodyRootElementNs, h.wsdlParser.GetTargetNamespace()) {
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
func (h *PluginHandler) calculateScore(exch *exchange.Exchange, reqMatcher *config.RequestMatcher, op *Operation, soapAction string) (score int, isWildcard bool) {
	// Get system XML namespaces
	var systemNamespaces map[string]string
	if h.config.System != nil {
		systemNamespaces = h.config.System.XMLNamespaces
	}

	// Calculate base score using common request matcher fields
	baseScore, baseWildcard := matcher.CalculateMatchScore(exch, reqMatcher, systemNamespaces, h.imposterConfig)
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
func (h *PluginHandler) HandleRequest(exch *exchange.Exchange, respProc response.Processor) {
	r := exch.Request.Request
	responseState := exch.ResponseState

	// Only handle POST requests for SOAP
	if r.Method != http.MethodPost {
		return
	}

	wsdlVersion := h.wsdlParser.GetVersion()

	// Parse the SOAP body first since we need it for both interceptors and resources
	bodyHolder, err := h.parseBody(exch.Request.Body)
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
		return
	}

	// Build SOAP-specific hooks that close over bodyHolder, op, and soapAction
	hooks := &pipeline.ProtocolHooks{
		CalculateScore: func(exch *exchange.Exchange, reqMatcher *config.RequestMatcher,
			systemNamespaces map[string]string, imposterConfig *config.ImposterConfig,
		) (int, bool) {
			return h.calculateScore(exch, reqMatcher, op, soapAction)
		},
		OnStepError: func(rs *exchange.ResponseState, msg string) {
			soapVersion := bodyHolder.GetSOAPVersion()
			h.failWithSOAPFault(soapVersion, bodyHolder.EnvNamespace, rs, msg, http.StatusInternalServerError)
			rs.Handled = true
		},
		ProcessResponse: func(exch *exchange.Exchange, reqMatcher *config.RequestMatcher,
			resp *config.Response, respProc response.Processor,
		) {
			h.processResponse(exch, bodyHolder, reqMatcher, resp, op, respProc)
		},
		GetResourceName: func(resource *config.Resource) (string, string) {
			return op.Name, "POST"
		},
	}

	pipeline.RunPipeline(h.config, h.imposterConfig, exch, respProc, hooks)
}
