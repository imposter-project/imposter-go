package websocket

import (
	"fmt"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/scheduler"
	"github.com/imposter-project/imposter-go/internal/steps"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// startSchedules launches connection-scoped schedules declared on the
// matched 'on: open' resource. They stop when the connection closes or when
// their firing limit is reached.
func (c *wsConn) startSchedules(resource *config.BaseResource) {
	if len(resource.Schedule) == 0 {
		return
	}
	logger.Debugf("starting %d connection-scoped schedule(s) - path:%s", len(resource.Schedule), c.upgrade.URL.Path)

	for i := range resource.Schedule {
		entry := &resource.Schedule[i]
		name := fmt.Sprintf("%s (connection %s)", scheduler.ScheduleName(entry, i), c.upgrade.URL.Path)
		next, err := scheduler.TriggerFunc(entry)
		if err != nil {
			// Validation runs at startup, so this should be unreachable.
			logger.Errorf("invalid trigger for schedule %s: %v", name, err)
			continue
		}

		limit := scheduler.EffectiveLimit(entry)
		logger.Debugf("registered schedule %s (%s)", name, scheduler.DescribeTrigger(entry, limit))

		c.wg.Add(1)
		go func(entry *config.Schedule, name string, next scheduler.NextFireFunc, limit int) {
			defer c.wg.Done()
			scheduler.RunSchedule(c.ctx, name, next, limit, func() {
				c.fireSchedule(entry, name)
			})
		}(entry, name, next, limit)
	}
}

// fireSchedule executes one firing of a connection-scoped schedule: any steps
// run first, then each response block is processed and sent as a text frame.
func (c *wsConn) fireSchedule(entry *config.Schedule, name string) {
	h := c.handler
	responseState := response.NewResponseState()
	exch := exchange.NewExchange(c.upgrade, nil, c.requestStore, responseState)
	reqMatcher := &config.RequestMatcher{}

	if len(entry.Steps) > 0 {
		if err := steps.RunSteps(entry.Steps, exch, h.imposterConfig, h.config.ConfigDir, responseState, reqMatcher); err != nil {
			logger.Errorf("schedule %s: failed to execute steps: %v", name, err)
			return
		}
	}

	var frames int
	resps := entry.EffectiveResponses()
	for i := range resps {
		if c.processAndSend(exch, reqMatcher, &resps[i], true) {
			frames++
		}
	}
	logger.Debugf("schedule %s: run completed (%d step(s), %d frame(s) sent)", name, len(entry.Steps), frames)
}
