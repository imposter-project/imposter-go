package openapi

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"net/http"
)

// preprocessResponse handles preparing the response state
func (h *PluginHandler) preprocessResponse(
	reqMatcher *config.RequestMatcher,
	rs *response.ResponseState,
	r *http.Request,
	resp *config.Response,
	requestStore *store.Store,
	preproc response.Processor,
) {
	if preproc != nil {
		preproc(reqMatcher, rs, r, resp, requestStore)
	}

	// Replace example placeholder in the response content
	if resp.Content == openapiExamplePlaceholder {
		openApiResp := h.lookupSpecResponse(r, *requestStore)
		if openApiResp == nil {
			logger.Errorf("no OpenAPI response with ID matched for request %s %s", r.Method, r.URL.Path)
			return
		}

		respHeaders, respContent := h.replaceExamplePlaceholder(rs.Headers, openApiResp)
		// note: this updates the config by reference, meaning the placeholder is replaced in the original config
		resp.Headers = respHeaders
		resp.Content = respContent
	}
}

// lookupSpecResponse gets the matched OpenAPI response for the request
func (h *PluginHandler) lookupSpecResponse(r *http.Request, requestStore store.Store) *Response {
	operationId := requestStore["_matched-openapi-operation"]
	if operationId == nil {
		logger.Tracef("no OpenAPI operation matched for request %s %s", r.Method, r.URL.Path)
		return nil
	}
	op := h.openApiParser.GetOperation(operationId.(string))

	responseId := requestStore["_matched-openapi-response"]
	if responseId == nil {
		// if an operation is matched, a response should also be matched
		logger.Errorf("no OpenAPI response matched for request %s %s", r.Method, r.URL.Path)
		return nil
	}

	return op.GetResponse(responseId.(string))
}

// replaceExamplePlaceholder replaces example placeholders in a template with a generated example response.
func (h *PluginHandler) replaceExamplePlaceholder(headers map[string]string, resp *Response) (respHeaders map[string]string, content string) {
	// Generate example response JSON
	exampleResponse, err := generateExampleJSON(resp.SparseResponse)
	if err != nil {
		logger.Warnf("failed to generate example body: %v", err)
		return nil, ""
	}

	// copy headers
	respHeaders = make(map[string]string)
	if headers != nil {
		for k, v := range headers {
			respHeaders[k] = v
		}
	}

	if resp.Headers != nil {
		for k, v := range resp.Headers {
			h, err := generateExampleString(v)
			if err != nil {
				logger.Warnf("failed to generate example header: %v", err)
				return nil, ""
			}
			respHeaders[k] = h
		}
	}
	return respHeaders, exampleResponse
}
