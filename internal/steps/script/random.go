package script

import (
	"github.com/dop251/goja"
	"github.com/imposter-project/imposter-go/internal/template"
)

// randomWrapper provides a JavaScript-friendly interface to random functions
type randomWrapper struct {
	runtime *goja.Runtime
}

// randomOptions holds parsed options for random functions
type randomOptions struct {
	length    int
	uppercase bool
	chars     string
}

func buildRandomWrapper(vm *goja.Runtime) map[string]interface{} {
	// Set up random object
	random := make(map[string]interface{})
	randomWrapper := &randomWrapper{runtime: vm}
	random["alphabetic"] = randomWrapper.alphabetic
	random["alphanumeric"] = randomWrapper.alphanumeric
	random["any"] = randomWrapper.any
	random["numeric"] = randomWrapper.numeric
	random["uuid"] = randomWrapper.uuid
	return random
}

// alphabetic generates a random alphabetic string
func (rw *randomWrapper) alphabetic(call goja.FunctionCall) goja.Value {
	options := parseRandomOptions(call, rw.runtime)
	// Use the function from template package
	return rw.runtime.ToValue(template.RandomAlphabetic(options.length, options.uppercase))
}

// alphanumeric generates a random alphanumeric string
func (rw *randomWrapper) alphanumeric(call goja.FunctionCall) goja.Value {
	options := parseRandomOptions(call, rw.runtime)
	// Use the function from template package
	return rw.runtime.ToValue(template.RandomAlphanumeric(options.length, options.uppercase))
}

// any generates a random string from a given character set
func (rw *randomWrapper) any(call goja.FunctionCall) goja.Value {
	options := parseRandomOptions(call, rw.runtime)
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Check if chars was provided as a parameter
	if options.chars != "" {
		chars = options.chars
	}

	// Use the function from template package
	return rw.runtime.ToValue(template.RandomAny(chars, options.length, options.uppercase))
}

// numeric generates a random numeric string
func (rw *randomWrapper) numeric(call goja.FunctionCall) goja.Value {
	options := parseRandomOptions(call, rw.runtime)
	// Use the function from template package
	return rw.runtime.ToValue(template.RandomNumeric(options.length))
}

// uuid generates a random UUID string
func (rw *randomWrapper) uuid(call goja.FunctionCall) goja.Value {
	options := parseRandomOptions(call, rw.runtime)
	// Use the function from template package
	return rw.runtime.ToValue(template.RandomUUID(options.uppercase))
}

// parseRandomOptions extracts options from JavaScript function call
func parseRandomOptions(call goja.FunctionCall, runtime *goja.Runtime) randomOptions {
	options := randomOptions{
		length:    1,     // Default length is 1
		uppercase: false, // Default is lowercase
		chars:     "",    // Default is empty (use function-specific charset)
	}

	// Check if an options object was provided
	if len(call.Arguments) > 0 && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		// Get the options object
		optsObj := call.Arguments[0].ToObject(runtime)

		// Get length option if provided
		lengthVal := optsObj.Get("length")
		if lengthVal != nil && !goja.IsUndefined(lengthVal) && !goja.IsNull(lengthVal) {
			options.length = int(lengthVal.ToInteger())
		}

		// Get uppercase option if provided
		uppercaseVal := optsObj.Get("uppercase")
		if uppercaseVal != nil && !goja.IsUndefined(uppercaseVal) && !goja.IsNull(uppercaseVal) {
			options.uppercase = uppercaseVal.ToBoolean()
		}

		// Get chars option if provided
		charsVal := optsObj.Get("chars")
		if charsVal != nil && !goja.IsUndefined(charsVal) && !goja.IsNull(charsVal) {
			options.chars = charsVal.String()
		}
	}

	return options
}
