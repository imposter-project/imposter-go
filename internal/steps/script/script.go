package script

import (
	"fmt"
	"os"

	"github.com/dop251/goja"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/imposter-project/imposter-go/pkg/utils"
)

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

// ExecuteScriptStep executes a script step
func ExecuteScriptStep(step *config.Step, exch *exchange.Exchange, imposterConfig *config.ImposterConfig, responseState *exchange.ResponseState, configDir string, reqMatcher *config.RequestMatcher) error {
	if step.Lang != "" && step.Lang != "js" && step.Lang != "javascript" {
		return fmt.Errorf("unsupported script language: %s", step.Lang)
	}

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

	vm := goja.New()

	console := buildConsole()
	vm.Set("console", console)

	if imposterConfig.LegacyConfigSupported {
		vm.Set("logger", console) // deprecated alias for console
	}

	vm.Set("context", buildContext(exch, reqMatcher))
	vm.Set("stores", buildStores(vm, exch))
	vm.Set("random", buildRandomWrapper(vm))
	vm.Set("respond", buildRespond(vm, responseState, imposterConfig))

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
