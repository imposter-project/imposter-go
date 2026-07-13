package websocket

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/scheduler"
	"github.com/imposter-project/imposter-go/internal/steps"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// startSchedules launches connection-scoped schedules declared on the
// matched 'on: open' resource. They stop when the connection closes.
func (c *wsConn) startSchedules(resource *config.BaseResource) {
	for i := range resource.Schedule {
		entry := &resource.Schedule[i]
		next, err := scheduler.TriggerFunc(entry)
		if err != nil {
			// Validation runs at startup, so this should be unreachable.
			logger.Errorf("invalid websocket schedule trigger - path:%s: %v", c.upgrade.URL.Path, err)
			continue
		}

		c.wg.Add(1)
		go func(entry *config.Schedule) {
			defer c.wg.Done()
			scheduler.RunSchedule(c.ctx, next, func() {
				c.fireSchedule(entry)
			})
		}(entry)
	}
}

// fireSchedule executes one firing of a connection-scoped schedule: any steps
// run first, then each response block is processed and sent as a text frame.
func (c *wsConn) fireSchedule(entry *config.Schedule) {
	h := c.handler
	responseState := response.NewResponseState()
	exch := exchange.NewExchange(c.upgrade, nil, c.requestStore, responseState)
	reqMatcher := &config.RequestMatcher{}

	if len(entry.Steps) > 0 {
		if err := steps.RunSteps(entry.Steps, exch, h.imposterConfig, h.config.ConfigDir, responseState, reqMatcher); err != nil {
			logger.Errorf("websocket schedule steps failed - path:%s: %v", c.upgrade.URL.Path, err)
			return
		}
	}

	resps := entry.EffectiveResponses()
	for i := range resps {
		c.processAndSend(exch, reqMatcher, &resps[i], true)
	}
}
