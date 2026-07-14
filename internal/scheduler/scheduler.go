package scheduler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/steps"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// Scheduler drives engine-lifetime scheduled jobs declared in top-level
// 'schedules' blocks. There is one scheduler per process.
type Scheduler struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

var (
	current *Scheduler
	mu      sync.Mutex
)

// Start launches a goroutine per schedule entry across all loaded configs.
// It is a no-op if no config declares schedules.
func Start(configs []*config.Config, imposterConfig *config.ImposterConfig) {
	mu.Lock()
	defer mu.Unlock()
	if current != nil {
		logger.Warnf("scheduler already started")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	sched := &Scheduler{cancel: cancel}

	var count int
	for _, cfg := range configs {
		for i := range cfg.Schedules {
			entry := &cfg.Schedules[i]
			name := ScheduleName(entry, i)
			next, err := TriggerFunc(entry)
			if err != nil {
				// Validation runs at startup, so this should be unreachable.
				logger.Errorf("invalid trigger for schedule %s: %v", name, err)
				continue
			}

			limit := EffectiveLimit(entry)
			logger.Infof("registered schedule %s (%s)", name, DescribeTrigger(entry, limit))
			if limit == 0 {
				logger.Infof("schedule %s has no 'limit' set and will keep firing for the lifetime of the engine; set 'limit' or %s to cap it", name, scheduleLimitEnvVar)
			} else if entry.Limit == 0 {
				logger.Infof("schedule %s: applying global default limit of %d from %s", name, limit, scheduleLimitEnvVar)
			}

			count++
			sched.wg.Add(1)
			go func(entry *config.Schedule, cfg *config.Config, name string, next NextFireFunc, limit int) {
				defer sched.wg.Done()
				RunSchedule(ctx, name, next, limit, func() {
					fireSchedule(entry, name, cfg, imposterConfig)
				})
			}(entry, cfg, name, next, limit)
		}
	}

	if count == 0 {
		cancel()
		return
	}

	current = sched
	logger.Infof("started %d scheduled job(s)", count)
}

// Stop cancels all schedule goroutines and waits for in-flight runs to finish.
func Stop() {
	mu.Lock()
	sched := current
	current = nil
	mu.Unlock()

	if sched == nil {
		return
	}
	sched.cancel()
	sched.wg.Wait()
}

// fireSchedule executes a single run of a schedule entry's steps with a fresh
// request-scoped store and response state.
func fireSchedule(entry *config.Schedule, name string, cfg *config.Config, imposterConfig *config.ImposterConfig) {
	startTime := time.Now()

	// Templating, delays and captures all dereference the exchange's request,
	// so a synthetic one is required even though no inbound request exists.
	req, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	if err != nil {
		logger.Errorf("schedule %s: failed to create synthetic request: %v", name, err)
		return
	}

	requestStore := store.NewRequestStore()
	responseState := response.NewResponseState()
	exch := exchange.NewExchange(req, nil, requestStore, responseState)

	if err := steps.RunSteps(entry.Steps, exch, imposterConfig, cfg.ConfigDir, responseState, &config.RequestMatcher{}); err != nil {
		logger.Errorf("schedule %s: failed to execute steps: %v", name, err)
		return
	}
	logger.Infof("schedule %s: run completed (%d step(s) in %v)", name, len(entry.Steps), time.Since(startTime).Round(time.Millisecond))
}
