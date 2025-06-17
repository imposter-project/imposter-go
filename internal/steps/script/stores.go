package script

import (
	"encoding/json"
	"github.com/dop251/goja"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/store"
)

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

func buildStores(vm *goja.Runtime, exch *exchange.Exchange) map[string]interface{} {
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
	return stores
}
