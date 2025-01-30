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
func (h *PluginHandler) sendSOAPFault(soapVersion SOAPVersion, envNamespace string, rs *response.ResponseState, message string, statusCode int) {
	rs.Headers["Content-Type"] = getResponseContentType(soapVersion)
	rs.StatusCode = statusCode

	var faultXML string
	if soapVersion == SOAP12 {
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

	faultXML = wrapInEnvelope(faultXML, envNamespace)
	rs.Body = []byte(faultXML)
}

// processResponse processes and sends the SOAP response
func (h *PluginHandler) processResponse(body *MessageBodyHolder, reqMatcher *config.RequestMatcher, rs *response.ResponseState, r *http.Request, resp config.Response, requestStore store.Store, op *Operation) {
	// Set content type for SOAP response
	rs.Headers["Content-Type"] = getResponseContentType(body.GetSOAPVersion())

	var finalResp config.Response
	if resp.StatusCode == http.StatusInternalServerError || resp.SoapFault {
		finalResp = h.applySoapFault(body, resp, op)
	} else {
		finalResp = resp
	}

	if finalResp.Content != "" {
		// Replace example placeholder in the response content
		finalResp.Content = h.replaceExamplePlaceholder(op, finalResp.Content, body)
	}

	// Process the response using common handler
	response.ProcessResponse(reqMatcher, rs, r, finalResp, h.configDir, requestStore, h.imposterConfig)
}

// applySoapFault applies a SOAP fault response if the response status code is 500 or if the response is marked as a SOAP fault.
func (h *PluginHandler) applySoapFault(body *MessageBodyHolder, resp config.Response, op *Operation) config.Response {
	finalResp := config.Response{
		StatusCode: http.StatusInternalServerError,
	}
	if resp.Content != "" {
		finalResp.Content = resp.Content
	} else if resp.File != "" {
		finalResp.File = resp.File
	} else {
		faultXml, err := generateExampleXML(op.Fault, h.wsdlParser.GetSchemaSystem())
		if err != nil {
			logger.Errorf("failed to generate example XML for fault: %v", err)
		}
		finalResp.Content = wrapInEnvelope(faultXml, body.EnvNamespace)
	}
	return finalResp
}

// replaceExamplePlaceholder replaces example placeholders in a template with a generated example response.
func (h *PluginHandler) replaceExamplePlaceholder(op *Operation, tmpl string, body *MessageBodyHolder) string {
	if tmpl != soapExamplePlaceholder {
		return tmpl
	}
	exampleXml, err := generateExampleXML(op.Output, h.wsdlParser.GetSchemaSystem())
	if err != nil {
		logger.Warnf("failed to generate example XML for operation %s: %v", op.Name, err)
		return ""
	}
	exampleResponse := wrapInEnvelope(exampleXml, body.EnvNamespace)
	return exampleResponse
}
