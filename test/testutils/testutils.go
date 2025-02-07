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
	rm.RequestHeaders = headers
	return config.Interceptor{
		RequestMatcher: rm,
		Response:       response,
		Continue:       cont,
	}
}

// NewInterceptorWithResponse creates a new Interceptor with response configuration for testing
func NewInterceptorWithResponse(method, path string, cont bool) config.Interceptor {
	rm := NewRequestMatcher(method, path)
	return config.Interceptor{
		RequestMatcher: rm,
		Continue:       cont,
	}
}
