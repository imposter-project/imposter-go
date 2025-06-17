package script

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/pkg/utils"
)

func buildContext(exch *exchange.Exchange, reqMatcher *config.RequestMatcher) map[string]interface{} {
	reqContext := make(map[string]interface{})
	reqContext["method"] = exch.Request.Request.Method
	reqContext["path"] = exch.Request.Request.URL.Path
	reqContext["uri"] = exch.Request.Request.URL.String()
	reqContext["body"] = string(exch.Request.Body)

	// Convert headers to a simple map
	headers := make(map[string]string)
	for k, v := range exch.Request.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	reqContext["headers"] = headers

	// Convert query parameters to a simple map
	queryParams := make(map[string]string)
	for k, v := range exch.Request.Request.URL.Query() {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}
	reqContext["queryParams"] = queryParams

	// Extract path parameters using the request matcher
	pathParams := make(map[string]string)
	if reqMatcher != nil && reqMatcher.Path != "" {
		pathParams = utils.ExtractPathParams(exch.Request.Request.URL.Path, reqMatcher.Path)
	}
	reqContext["pathParams"] = pathParams

	// Parse and convert form parameters to a simple map
	formParams := make(map[string]string)
	if err := exch.Request.Request.ParseForm(); err == nil {
		for k, v := range exch.Request.Request.PostForm {
			if len(v) > 0 {
				formParams[k] = v[0]
			}
		}
	}
	reqContext["formParams"] = formParams

	// Set up context object
	context := make(map[string]interface{})
	context["request"] = reqContext
	return context
}
