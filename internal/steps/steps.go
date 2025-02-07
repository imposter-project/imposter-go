package steps

import (
	"fmt"
	"net/http"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// Context holds the context for step execution
type Context struct {
	Request      *http.Request
	RequestBody  []byte
	RequestStore *store.Store
}

// RunSteps executes a sequence of steps in order
func RunSteps(steps []config.Step, ctx *Context) error {
	for i, step := range steps {
		var err error
		switch step.Type {
		case config.ScriptStepType:
			err = executeScriptStep(&step, ctx)
		case config.RemoteStepType:
			err = executeRemoteStep(&step, ctx)
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
