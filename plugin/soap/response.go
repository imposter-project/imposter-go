package soap

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"net/http"
)

// failWithSOAPFault fails the request and sends a SOAP fault response
func (h *PluginHandler) failWithSOAPFault(soapVersion SOAPVersion, envNamespace string, rs *response.ResponseState, message string, statusCode int) {
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
func (h *PluginHandler) processResponse(
	body *MessageBodyHolder,
	reqMatcher *config.RequestMatcher,
	rs *response.ResponseState,
	r *http.Request,
	resp *config.Response,
	requestStore *store.Store,
	op *Operation,
	respProc response.Processor,
) {
	// Set content type for SOAP response
	rs.Headers["Content-Type"] = getResponseContentType(body.GetSOAPVersion())

	var finalResp *config.Response
	if resp.StatusCode == http.StatusInternalServerError || resp.SoapFault {
		soapResp := h.generateSoapFault(body, resp, op)
		finalResp = &soapResp
	} else {
		finalResp = resp
	}

	// Replace example placeholder in the response content
	if finalResp.Content == soapExamplePlaceholder {
		// note: this updates the config by reference, meaning the placeholder is replaced in the original config
		finalResp.Content = h.replaceExamplePlaceholder(op, body)
	}

	// Standard response processor
	respProc(reqMatcher, rs, r, finalResp, requestStore)
}

// generateSoapFault applies a SOAP fault response if the response status code is 500 or if the response is marked as a SOAP fault.
func (h *PluginHandler) generateSoapFault(body *MessageBodyHolder, resp *config.Response, op *Operation) config.Response {
	var statusCode int
	if resp.StatusCode != 0 {
		statusCode = resp.StatusCode
	} else {
		statusCode = http.StatusInternalServerError
	}
	faultResp := config.Response{
		StatusCode: statusCode,
		Delay:      resp.Delay,
		Fail:       resp.Fail,
	}
	if resp.Content != "" {
		faultResp.Content = resp.Content
	} else if resp.File != "" {
		faultResp.File = resp.File
	} else {
		faultXml, err := generateExampleXML(op.Fault, h.wsdlParser.GetSchemaSystem())
		if err != nil {
			logger.Errorf("failed to generate example XML for fault: %v", err)
		}
		faultResp.Content = wrapInEnvelope(faultXml, body.EnvNamespace)
	}
	return faultResp
}

// replaceExamplePlaceholder replaces example placeholders in a template with a generated example response.
func (h *PluginHandler) replaceExamplePlaceholder(op *Operation, body *MessageBodyHolder) string {
	exampleXml, err := generateExampleXML(op.Output, h.wsdlParser.GetSchemaSystem())
	if err != nil {
		logger.Warnf("failed to generate example XML for operation %s: %v", op.Name, err)
		return ""
	}
	exampleResponse := wrapInEnvelope(exampleXml, body.EnvNamespace)
	return exampleResponse
}
