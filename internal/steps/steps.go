package steps

import (
	"fmt"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// RunSteps executes a sequence of steps in order
func RunSteps(steps []config.Step, exch *exchange.Exchange, imposterConfig *config.ImposterConfig, configDir string, responseState *exchange.ResponseState, reqMatcher *config.RequestMatcher) error {
	for i, step := range steps {
		var err error
		switch step.Type {
		case config.ScriptStepType:
			err = executeScriptStep(&step, exch, imposterConfig, responseState, configDir, reqMatcher)
		case config.RemoteStepType:
			err = executeRemoteStep(&step, exch, imposterConfig)
		default:
			err = fmt.Errorf("unknown step type: %s", step.Type)
		}

		if err != nil {
			logger.Errorf("failed to execute step %d: %v", i+1, err)
			return err
		}
	}
	return nil
}
