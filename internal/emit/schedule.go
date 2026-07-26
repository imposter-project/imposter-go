package emit

import (
	"context"
	"fmt"
	"sync"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/scheduler"
	"github.com/imposter-project/imposter-go/internal/steps"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// ScheduleHost runs a set of scoped schedules — connection-scoped (a websocket
// connection) or request-scoped (a streamed HTTP response) — firing each on
// its trigger until the context is cancelled or its firing limit is reached.
// Each firing runs the entry's steps, then emits its responses via the sink.
//
// This is the generalisation of the websocket plugin's connection-scoped
// schedules: only NewExchange and Sink differ between protocols.
type ScheduleHost struct {
	// Ctx bounds every schedule's lifetime; cancelling it stops all of them
	// (e.g. the websocket connection closed, or the HTTP client disconnected).
	Ctx context.Context

	// Schedules are the entries to run.
	Schedules []config.Schedule

	// Label identifies this host in log messages, e.g. "connection /ws" or
	// "request /events".
	Label string

	ImposterConfig *config.ImposterConfig
	ConfigDir      string
	RespProc       response.Processor
	Sink           Sink

	// NewExchange builds a fresh exchange for a single firing. Each firing gets
	// its own response state so concurrent or successive firings do not clash;
	// the protocol supplies the request and request-scoped store that captures
	// and templates dereference.
	NewExchange func() *exchange.Exchange
}

// Start launches one goroutine per schedule, registering each on wg. It returns
// immediately; callers that must not outlive the schedules (e.g. an HTTP
// handler holding the response open) should wg.Wait().
func (h *ScheduleHost) Start(wg *sync.WaitGroup) {
	if len(h.Schedules) == 0 {
		return
	}
	logger.Debugf("starting %d scoped schedule(s) - %s", len(h.Schedules), h.Label)

	for i := range h.Schedules {
		entry := &h.Schedules[i]
		name := fmt.Sprintf("%s (%s)", scheduler.ScheduleName(entry, i), h.Label)
		next, err := scheduler.TriggerFunc(entry)
		if err != nil {
			// Validation runs at startup, so this should be unreachable.
			logger.Errorf("invalid trigger for schedule %s: %v", name, err)
			continue
		}

		limit := scheduler.EffectiveLimit(entry)
		logger.Debugf("registered schedule %s (%s)", name, scheduler.DescribeTrigger(entry, limit))

		wg.Add(1)
		go func(entry *config.Schedule, name string, next scheduler.NextFireFunc, limit int) {
			defer wg.Done()
			scheduler.RunSchedule(h.Ctx, name, next, limit, func() {
				h.fire(entry, name)
			})
		}(entry, name, next, limit)
	}
}

// fire executes one firing: run the entry's steps, then emit each of its
// responses via the sink.
func (h *ScheduleHost) fire(entry *config.Schedule, name string) {
	exch := h.NewExchange()
	reqMatcher := &config.RequestMatcher{}

	if len(entry.Steps) > 0 {
		if err := steps.RunSteps(entry.Steps, exch, h.ImposterConfig, h.ConfigDir, exch.ResponseState, reqMatcher); err != nil {
			logger.Errorf("schedule %s: failed to execute steps: %v", name, err)
			return
		}
	}

	emitted := EmitResponses(exch, reqMatcher, entry.EffectiveResponses(), h.RespProc, h.Sink)
	logger.Debugf("schedule %s: run completed (%d step(s), %d response(s) emitted)", name, len(entry.Steps), emitted)
}
