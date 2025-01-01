package testutils

import (
	"github.com/imposter-project/imposter-go/internal/config"
)

// NewRequestMatcher creates a new RequestMatcher for testing
func NewRequestMatcher(method, path string) config.RequestMatcher {
	return config.RequestMatcher{
		Method: method,
		Path:   path,
	}
}

// NewResource creates a new Resource for testing
func NewResource(method, path string, response config.Response) config.Resource {
	return config.Resource{
		RequestMatcher: NewRequestMatcher(method, path),
		Response:       response,
	}
}

// NewInterceptor creates a new Interceptor for testing
func NewInterceptor(method, path string, headers map[string]config.MatcherUnmarshaler, response *config.Response, cont bool) config.Interceptor {
	rm := NewRequestMatcher(method, path)
	rm.Headers = headers
	return config.Interceptor{
		RequestMatcher: rm,
		Response:       response,
		Continue:       cont,
	}
}

// NewInterceptorWithCapture creates a new Interceptor with capture configuration for testing
func NewInterceptorWithCapture(method, path string, capture map[string]config.Capture, cont bool) config.Interceptor {
	rm := NewRequestMatcher(method, path)
	rm.Capture = capture
	return config.Interceptor{
		RequestMatcher: rm,
		Continue:       cont,
	}
}
