package interceptor

import (
	"net/http"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
)

// ProcessInterceptor handles an interceptor and returns true if request processing should continue
func ProcessInterceptor(reqMatcher *config.RequestMatcher, rs *response.ResponseState, r *http.Request, body []byte, interceptor config.Interceptor, requestStore store.Store, imposterConfig *config.ImposterConfig, processor response.Processor) bool {
	// Capture request data if specified
	if interceptor.Capture != nil {
		capture.CaptureRequestData(imposterConfig, interceptor.Capture, r, body, requestStore)
	}

	if interceptor.Response != nil {
		processor(reqMatcher, rs, r, *interceptor.Response, requestStore)
	}

	return interceptor.Continue
}
