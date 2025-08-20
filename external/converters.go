package external

import (
	"github.com/imposter-project/imposter-go/external/shared"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"net/http"
)

func ConvertToExternalRequest(req *http.Request) shared.HandlerRequest {
	return shared.HandlerRequest{
		Method:  req.Method,
		Path:    req.URL.Path,
		Headers: nil,
	}
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
