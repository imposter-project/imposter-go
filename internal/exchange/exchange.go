package exchange

import (
	"net/http"

	"github.com/imposter-project/imposter-go/internal/store"
)

// Exchange holds the data for a request/response exchange
type Exchange struct {
	Request       *RequestContext
	RequestStore  *store.Store
	Response      *ResponseContext
	ResponseState *ResponseState
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

// NewExchange creates a new Exchange object from an HTTP request, body, request store, and response state
func NewExchange(req *http.Request, body []byte, requestStore *store.Store, responseState *ResponseState) *Exchange {
	return &Exchange{
		Request: &RequestContext{
			Request: req,
			Body:    body,
		},
		RequestStore:  requestStore,
		ResponseState: responseState,
	}
}

// NewExchangeFromRequest creates a new Exchange object from an HTTP request, body, and request store
func NewExchangeFromRequest(req *http.Request, body []byte, requestStore *store.Store) *Exchange {
	return NewExchange(req, body, requestStore, nil)
}
