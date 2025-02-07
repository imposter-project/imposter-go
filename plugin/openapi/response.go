package openapi

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/imposter-project/imposter-go/pkg/utils"
	"net/http"
)

// processResponse handles preparing the response state
func (h *PluginHandler) processResponse(
	reqMatcher *config.RequestMatcher,
	rs *response.ResponseState,
	r *http.Request,
	resp *config.Response,
	requestStore *store.Store,
	respProc response.Processor,
) {
	// Replace example placeholder in the response content
	if resp.Content == openapiExamplePlaceholder || resp.ExampleName != "" {
		specResp := h.lookupSpecResponse(r, *requestStore)
		if specResp == nil {
			logger.Errorf("no OpenAPI response with ID matched for request %s %s", r.Method, r.URL.Path)
			return
		}

		respHeaders, respContent := h.replaceExamplePlaceholder(rs.Headers, specResp, resp)
		// note: this updates the config by reference, meaning the placeholder is replaced in the original config
		resp.Headers = respHeaders
		resp.Content = respContent
	}

	// Standard response processor
	respProc(reqMatcher, rs, r, resp, requestStore)
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
func (h *PluginHandler) replaceExamplePlaceholder(headers map[string]string, specResp *Response, resp *config.Response) (respHeaders map[string]string, content string) {
	// check if an example name is provided in the response config or OpenAPI spec, otherwise
	// will fall back to generation from schema
	exampleName := checkForExample(resp, specResp)

	// Generate example response JSON
	exampleResponse, err := generateExampleJSON(specResp.SparseResponse, exampleName)
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

	if specResp.Headers != nil {
		for k, v := range specResp.Headers {
			h, err := generateExampleString(v, defaultExampleName)
			if err != nil {
				logger.Warnf("failed to generate example header: %v", err)
				return nil, ""
			}
			respHeaders[k] = h
		}
	}
	return respHeaders, exampleResponse
}

// checkForExample checks if an example name is provided in the response config or OpenAPI spec
func checkForExample(resp *config.Response, specResp *Response) string {
	var exampleName string
	if resp.ExampleName != "" {
		exampleName = resp.ExampleName

	} else if specResp.Examples != nil && len(specResp.Examples) > 0 {
		// if there are one or more examples in the OpenAPI spec, use the first one
		_, exists := specResp.Examples[defaultExampleName]
		if exists {
			exampleName = defaultExampleName
		} else {
			exName, _ := utils.GetFirstItemFromMap(specResp.Examples)
			exampleName = exName
		}
	}
	return exampleName
}
