package exchange

import (
	"net/http"

	"github.com/imposter-project/imposter-go/internal/store"
)

// Exchange holds the data for a request/response exchange
type Exchange struct {
	Request      *RequestContext
	Response     *ResponseContext
	RequestStore *store.Store
}

// RequestContext holds request-related data
type RequestContext struct {
	Request *http.Request
	Body    []byte
}

// ResponseContext holds response-related data
type ResponseContext struct {
	Response *http.Response
	Body     []byte
}

func NewExchangeFromRequest(r *http.Request, body []byte, requestStore *store.Store) *Exchange {
	return &Exchange{
		Request: &RequestContext{
			Request: r,
			Body:    body,
		},
		RequestStore: requestStore,
	}
}
