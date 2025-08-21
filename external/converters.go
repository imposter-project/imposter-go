package external

import (
	"github.com/imposter-project/imposter-go/external/shared"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"net/http"
)

func ConvertToExternalRequest(exch *exchange.Exchange) shared.HandlerRequest {
	req := exch.Request.Request
	return shared.HandlerRequest{
		Method:  req.Method,
		Path:    req.URL.Path,
		Query:   req.URL.Query(),
		Headers: convertToSingleValueHeaders(req.Header),
		Body:    exch.Request.Body,
	}
}

func convertToSingleValueHeaders(header http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range header {
		if len(values) > 0 {
			// Use the first value if multiple values exist
			result[key] = values[0]
		}
	}
	return result
}

func ConvertFromExternalResponse(exch *exchange.Exchange, handlerResp *shared.HandlerResponse) {
	rs := exch.ResponseState
	rs.Handled = true
	rs.StatusCode = handlerResp.StatusCode
	rs.File = handlerResp.File
	rs.Body = handlerResp.Body

	response.CopyResponseHeaders(handlerResp.Headers, rs)
	response.SetContentTypeHeader(rs, handlerResp.FileName, "", "")
}
