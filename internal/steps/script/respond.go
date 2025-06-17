package script

import (
	"github.com/dop251/goja"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
)

// ResponseBuilder provides a fluent API for building responses in scripts
type ResponseBuilder struct {
	runtime *goja.Runtime
	state   *exchange.ResponseState
	obj     *goja.Object
}

func (rb *ResponseBuilder) withStatusCode(statusCode int) goja.Value {
	rb.state.StatusCode = statusCode
	return rb.obj
}

func (rb *ResponseBuilder) withContent(content string) goja.Value {
	rb.state.Body = []byte(content)
	return rb.obj
}

func (rb *ResponseBuilder) withFile(filePath string) goja.Value {
	// Set the file path on the response state
	rb.state.File = filePath
	return rb.obj
}

func (rb *ResponseBuilder) withHeader(name, value string) goja.Value {
	if rb.state.Headers == nil {
		rb.state.Headers = make(map[string]string)
	}
	rb.state.Headers[name] = value
	return rb.obj
}

func (rb *ResponseBuilder) withEmpty() goja.Value {
	rb.state.Body = []byte{}
	return rb.obj
}

func (rb *ResponseBuilder) withDelay(exactDelay int) goja.Value {
	rb.state.Delay.Exact = exactDelay
	rb.state.Delay.Min = 0
	rb.state.Delay.Max = 0
	return rb.obj
}

func (rb *ResponseBuilder) withDelayRange(minDelay, maxDelay int) goja.Value {
	rb.state.Delay.Exact = 0
	rb.state.Delay.Min = minDelay
	rb.state.Delay.Max = maxDelay
	return rb.obj
}

func (rb *ResponseBuilder) withFailure(failureType string) goja.Value {
	rb.state.Fail = failureType
	return rb.obj
}

func buildRespond(vm *goja.Runtime, responseState *exchange.ResponseState, imposterConfig *config.ImposterConfig) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		obj := vm.NewObject()
		rb := &ResponseBuilder{
			runtime: vm,
			state:   responseState,
			obj:     obj,
		}
		_ = obj.Set("withStatusCode", rb.withStatusCode)
		_ = obj.Set("withContent", rb.withContent)
		_ = obj.Set("withFile", rb.withFile)
		_ = obj.Set("withHeader", rb.withHeader)
		_ = obj.Set("withEmpty", rb.withEmpty)
		_ = obj.Set("withDelay", rb.withDelay)
		_ = obj.Set("withDelayRange", rb.withDelayRange)
		_ = obj.Set("withFailure", rb.withFailure)
		_ = obj.Set("usingDefaultBehaviour", rb.usingDefaultBehaviour)
		_ = obj.Set("skipDefaultBehaviour", rb.skipDefaultBehaviour)
		_ = obj.Set("and", rb.and)

		// legacy functions
		if imposterConfig.LegacyConfigSupported {
			_ = obj.Set("withData", rb.withContent)
		}

		return obj
	}
}
