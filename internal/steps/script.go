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
)

// storeWrapper provides a JavaScript-friendly interface to a store
type storeWrapper struct {
	runtime *goja.Runtime
	name    string
}

func (sw *storeWrapper) save(key string, value interface{}) {
	store.StoreValue(sw.name, key, value)
}

func (sw *storeWrapper) load(key string) interface{} {
	val, found := store.GetValue(sw.name, key)
	if !found {
		return nil
	}
	return val
}

func (sw *storeWrapper) loadAsJson(key string) interface{} {
	val, found := store.GetValue(sw.name, key)
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
	store.DeleteValue(sw.name, key)
}

func (sw *storeWrapper) loadAll() interface{} {
	return store.GetAllValues(sw.name, "") // Empty string means no prefix filter
}

func (sw *storeWrapper) hasItemWithKey(key string) bool {
	_, found := store.GetValue(sw.name, key)
	return found
}

// executeScriptStep executes a script step
func executeScriptStep(step *config.Step, exch *exchange.Exchange) error {
	// Validate step configuration
	if step.Lang != "" && step.Lang != "javascript" {
		return fmt.Errorf("unsupported script language: %s", step.Lang)
	}

	// Get script content
	var scriptContent string
	if step.Code != "" {
		scriptContent = step.Code
	} else if step.File != "" {
		content, err := os.ReadFile(step.File)
		if err != nil {
			return fmt.Errorf("failed to read script file %s: %w", step.File, err)
		}
		scriptContent = string(content)
	} else {
		return fmt.Errorf("either code or file must be specified for script step")
	}

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
			name:    storeName,
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

	// Run the script
	_, err := vm.RunString(scriptContent)
	if err != nil {
		if jsErr, ok := err.(*goja.Exception); ok {
			return fmt.Errorf("script execution failed: %v", jsErr.Value())
		}
		return fmt.Errorf("script execution failed: %w", err)
	}

	return nil
}
