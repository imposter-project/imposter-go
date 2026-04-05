package external

import (
	"github.com/imposter-project/imposter-go/external/shared"
	"github.com/imposter-project/imposter-go/internal/exchange"
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
