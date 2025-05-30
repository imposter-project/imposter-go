package steps

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dop251/goja"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/imposter-project/imposter-go/pkg/utils"
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

func (rb *ResponseBuilder) usingDefaultBehaviour() goja.Value {
	rb.state.Handled = false
	return rb.obj
}

func (rb *ResponseBuilder) skipDefaultBehaviour() goja.Value {
	rb.state.Handled = true
	return rb.obj
}

func (rb *ResponseBuilder) and() goja.Value {
	return rb.obj
}

// storeWrapper provides a JavaScript-friendly interface to a store
type storeWrapper struct {
	runtime *goja.Runtime
	store   *store.Store
}

func (sw *storeWrapper) save(key string, value interface{}) {
	sw.store.StoreValue(key, value)
}

func (sw *storeWrapper) load(key string) interface{} {
	val, found := sw.store.GetValue(key)
	if !found {
		return nil
	}
	return val
}

func (sw *storeWrapper) loadAsJson(key string) interface{} {
	val, found := sw.store.GetValue(key)
	if !found {
		return nil
	}

	// If it's already a string, try to parse it as JSON
	if str, ok := val.(string); ok {
		var jsonData interface{}
		if err := json.Unmarshal([]byte(str), &jsonData); err == nil {
			return sw.runtime.ToValue(jsonData)
		}
	}

	// Otherwise, convert the value to JSON and back to ensure proper type conversion
	jsonBytes, err := json.Marshal(val)
	if err != nil {
		return nil
	}

	var jsonData interface{}
	if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
		return nil
	}
	return sw.runtime.ToValue(jsonData)
}

func (sw *storeWrapper) delete(key string) {
	sw.store.DeleteValue(key)
}

func (sw *storeWrapper) loadAll() interface{} {
	return sw.store.GetAllValues("") // Empty string means no prefix filter
}

func (sw *storeWrapper) hasItemWithKey(key string) bool {
	_, found := sw.store.GetValue(key)
	return found
}

// executeScriptStep executes a script step
func executeScriptStep(step *config.Step, exch *exchange.Exchange, imposterConfig *config.ImposterConfig, responseState *exchange.ResponseState, configDir string, reqMatcher *config.RequestMatcher) error {
	// Validate step configuration
	if step.Lang != "" && step.Lang != "js" && step.Lang != "javascript" {
		return fmt.Errorf("unsupported script language: %s", step.Lang)
	}

	// Get script content
	var scriptContent string
	if step.Code != "" {
		scriptContent = step.Code
		logger.Infoln("executing inline script")
	} else if step.File != "" {
		scriptFile, err := utils.ValidatePath(step.File, configDir)
		if err != nil {
			return fmt.Errorf("failed to validate script file path: %w", err)
		}
		content, err := os.ReadFile(scriptFile)
		if err != nil {
			return fmt.Errorf("failed to read script file %s: %w", scriptFile, err)
		}
		scriptContent = string(content)
		logger.Infof("executing script from file %s", step.File)
	} else {
		return fmt.Errorf("either code or file must be specified for script step")
	}

	logger.Tracef("script content: %s", scriptContent)

	// Create JavaScript runtime
	vm := goja.New()

	// Set up console.log
	console := make(map[string]interface{})
	console["log"] = func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			logger.Infof("[Script] %v", arg)
		}
		return goja.Undefined()
	}
	vm.Set("console", console)

	// Make request context available to script
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
	vm.Set("context", context)

	// Set up stores object with open function
	stores := make(map[string]interface{})
	stores["open"] = func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("store name must be provided"))
		}
		storeName := call.Arguments[0].String()
		wrapper := &storeWrapper{
			runtime: vm,
			store:   store.Open(storeName, exch.RequestStore),
		}

		obj := vm.NewObject()
		_ = obj.Set("save", wrapper.save)
		_ = obj.Set("load", wrapper.load)
		_ = obj.Set("loadAsJson", wrapper.loadAsJson)
		_ = obj.Set("delete", wrapper.delete)
		_ = obj.Set("loadAll", wrapper.loadAll)
		_ = obj.Set("hasItemWithKey", wrapper.hasItemWithKey)

		return obj
	}
	vm.Set("stores", stores)

	// Set up respond function
	vm.Set("respond", func(call goja.FunctionCall) goja.Value {
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
	})

	// Run the script
	_, err := vm.RunString(scriptContent)
	if err != nil {
		if jsErr, ok := err.(*goja.Exception); ok {
			return fmt.Errorf("script execution failed: %v", jsErr.Value())
		}
		return fmt.Errorf("script execution failed: %w", err)
	}

	logger.Debugln("script execution completed successfully")
	return nil
}
