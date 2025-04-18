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
func NewResource(method, path string, response *config.Response) config.Resource {
	return config.Resource{
		BaseResource: config.BaseResource{
			RequestMatcher: NewRequestMatcher(method, path),
			Response:       response,
		},
	}
}

// NewResourceWithLog creates a new Resource with a log message for testing
func NewResourceWithLog(method, path string, response *config.Response, logMessage string) config.Resource {
	return config.Resource{
		BaseResource: config.BaseResource{
			RequestMatcher: NewRequestMatcher(method, path),
			Response:       response,
			Log:            logMessage,
		},
	}
}

// NewInterceptor creates a new Interceptor for testing
func NewInterceptor(method, path string, headers map[string]config.MatcherUnmarshaler, response *config.Response, cont bool) config.Interceptor {
	rm := NewRequestMatcher(method, path)
	rm.RequestHeaders = headers
	return config.Interceptor{
		BaseResource: config.BaseResource{
			RequestMatcher: rm,
			Response:       response,
		},
		Continue: cont,
	}
}

// NewInterceptorWithLog creates a new Interceptor with a log message for testing
func NewInterceptorWithLog(method, path string, headers map[string]config.MatcherUnmarshaler, response *config.Response, cont bool, logMessage string) config.Interceptor {
	rm := NewRequestMatcher(method, path)
	rm.RequestHeaders = headers
	return config.Interceptor{
		BaseResource: config.BaseResource{
			RequestMatcher: rm,
			Response:       response,
			Log:            logMessage,
		},
		Continue: cont,
	}
}

// NewInterceptorWithResponse creates a new Interceptor with response configuration for testing
func NewInterceptorWithResponse(method, path string, cont bool) config.Interceptor {
	rm := NewRequestMatcher(method, path)
	return config.Interceptor{
		BaseResource: config.BaseResource{
			RequestMatcher: rm,
		},
		Continue: cont,
	}
}
