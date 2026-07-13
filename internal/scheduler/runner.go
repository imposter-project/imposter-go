package scheduler

import (
	"context"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/robfig/cron/v3"
)

// NextFireFunc returns the next fire time after the given time. It abstracts
// the trigger type (interval or cron) so both share one runner loop.
type NextFireFunc func(after time.Time) time.Time

// TriggerFunc builds the NextFireFunc for a schedule entry. Validation has
// already ensured exactly one of Every or Cron is set and parses.
func TriggerFunc(sched *config.Schedule) (NextFireFunc, error) {
	if sched.Every != "" {
		interval, err := time.ParseDuration(sched.Every)
		if err != nil {
			return nil, err
		}
		return func(after time.Time) time.Time {
			return after.Add(interval)
		}, nil
	}

	cronSched, err := cron.ParseStandard(sched.Cron)
	if err != nil {
		return nil, err
	}
	return cronSched.Next, nil
}

// RunSchedule fires the given function according to the schedule's trigger
// until the context is cancelled. Runs are serialised: a firing that takes
// longer than the interval delays the next one rather than overlapping it.
func RunSchedule(ctx context.Context, next NextFireFunc, fire func()) {
	timer := time.NewTimer(time.Until(next(time.Now())))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			fire()
			timer.Reset(time.Until(next(time.Now())))
		case <-ctx.Done():
			return
		}
	}
}
