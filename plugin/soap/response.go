package soap

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"net/http"
)

// sendSOAPFault sends a SOAP fault response
func (h *PluginHandler) sendSOAPFault(rs *response.ResponseState, message string, statusCode int) {
	rs.Headers["Content-Type"] = getResponseContentType(h.wsdlParser.GetSOAPVersion())
	rs.StatusCode = statusCode

	var faultXML string
	if h.wsdlParser.GetSOAPVersion() == SOAP12 {
		faultXML = fmt.Sprintf(`<env:Fault>
            <env:Code>
                <env:Value>env:Receiver</env:Value>
            </env:Code>
            <env:Reason>
                <env:Text>%s</env:Text>
            </env:Reason>
        </env:Fault>`, message)
	} else {
		faultXML = fmt.Sprintf(`<env:Fault>
            <faultcode>env:Server</faultcode>
            <faultstring>%s</faultstring>
        </env:Fault>`, message)
	}

	faultXML = wrapInEnvelope(faultXML, h.wsdlParser.GetSOAPVersion())
	rs.Body = []byte(faultXML)
}

// processResponse processes and sends the SOAP response
func (h *PluginHandler) processResponse(reqMatcher *config.RequestMatcher, rs *response.ResponseState, r *http.Request, resp config.Response, requestStore store.Store, op *Operation) {
	// Set content type for SOAP response
	rs.Headers["Content-Type"] = getResponseContentType(h.wsdlParser.GetSOAPVersion())

	// Handle SOAP faults from config
	var finalResp config.Response
	if resp.StatusCode == http.StatusInternalServerError || resp.SoapFault {
		finalResp = config.Response{
			StatusCode: http.StatusInternalServerError,
		}
		if resp.Content != "" {
			finalResp.Content = resp.Content
		} else if resp.File != "" {
			finalResp.File = resp.File
		} else {
			// TODO handle fault types as well as elements
			faultXml, err := generateExampleXML(op.Fault, h.wsdlParser.GetSchemaSystem())
			if err != nil {
				logger.Errorf("failed to generate example XML for fault: %v", err)
			}
			finalResp.Content = wrapInEnvelope(faultXml, h.wsdlParser.GetSOAPVersion())
		}
	} else {
		finalResp = resp
	}

	// Process the response using common handler
	response.ProcessResponse(reqMatcher, rs, r, finalResp, h.configDir, requestStore, h.imposterConfig)
}

// wrapInEnvelope wraps the given XML content in a SOAP envelope,
// using the specified SOAP version. The prefix "env" is used for the envelope
// namespace.
func wrapInEnvelope(content string, version SOAPVersion) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<env:Envelope xmlns:env="%s">
  <env:Body>%s</env:Body>
</env:Envelope>`, getEnvNamespace(version), content)
}

// getResponseContentType returns the content type for the SOAP response
// based on the SOAP version
func getResponseContentType(soapVersion SOAPVersion) string {
	switch {
	case soapVersion == SOAP12:
		return "application/soap+xml"
	default:
		return "text/xml"
	}
}
