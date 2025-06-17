package script

import (
	"github.com/dop251/goja"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

func buildConsole() map[string]interface{} {
	console := make(map[string]interface{})
	console["log"] = func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			logger.Infof("[Script] %v", arg)
		}
		return goja.Undefined()
	}
	return console
}
