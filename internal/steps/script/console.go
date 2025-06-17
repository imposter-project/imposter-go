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

	console["debug"] = func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			logger.Debugf("[Script] %v", arg)
		}
		return goja.Undefined()
	}

	console["error"] = func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			logger.Errorf("[Script] %v", arg)
		}
		return goja.Undefined()
	}

	console["warn"] = func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			logger.Warnf("[Script] %v", arg)
		}
		return goja.Undefined()
	}

	console["info"] = func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			logger.Infof("[Script] %v", arg)
		}
		return goja.Undefined()
	}

	return console
}
