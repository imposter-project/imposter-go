package response

import (
	"net/http"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
)

// Processor is a function that plugins implement to handle their specific response processing
type Processor func(reqMatcher *config.RequestMatcher, rs *ResponseState, r *http.Request, resp config.Response, requestStore store.Store)
