package scheduler

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/robfig/cron/v3"
)

// scheduleLimitEnvVar is the environment variable holding a global default
// firing limit, applied to any schedule that does not set 'limit' itself.
const scheduleLimitEnvVar = "IMPOSTER_SCHEDULE_LIMIT"

// EffectiveLimit returns the schedule's firing limit: the schedule's own
// 'limit' when set, otherwise the global default from IMPOSTER_SCHEDULE_LIMIT,
// otherwise 0 (unlimited).
func EffectiveLimit(sched *config.Schedule) int {
	if sched.Limit > 0 {
		return sched.Limit
	}
	return globalDefaultLimit()
}

// globalDefaultLimit reads the operator-set default firing limit from the
// environment, returning 0 (unlimited) when unset or invalid.
func globalDefaultLimit() int {
	raw := os.Getenv(scheduleLimitEnvVar)
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		logger.Warnf("invalid %s %q; ignoring (must be a non-negative integer)", scheduleLimitEnvVar, raw)
		return 0
	}
	return parsed
}

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

// DescribeTrigger renders a schedule's trigger and effective limit for log
// messages, e.g. "every 30s, limit 10" or "cron '0 * * * *', unlimited".
func DescribeTrigger(sched *config.Schedule, effectiveLimit int) string {
	var trigger string
	if sched.Every != "" {
		trigger = fmt.Sprintf("every %s", sched.Every)
	} else {
		trigger = fmt.Sprintf("cron '%s'", sched.Cron)
	}
	if effectiveLimit > 0 {
		return fmt.Sprintf("%s, limit %d", trigger, effectiveLimit)
	}
	return fmt.Sprintf("%s, unlimited", trigger)
}

// ScheduleName returns a human-readable identifier for a schedule entry,
// falling back to its index when unnamed.
func ScheduleName(sched *config.Schedule, idx int) string {
	if sched.Name != "" {
		return fmt.Sprintf("%q", sched.Name)
	}
	return fmt.Sprintf("at index %d", idx)
}

// RunSchedule fires the given function according to the schedule's trigger
// until the context is cancelled or the firing limit (if positive) is
// reached. Runs are serialised: a firing that takes longer than the interval
// delays the next one rather than overlapping it.
func RunSchedule(ctx context.Context, name string, next NextFireFunc, limit int, fire func()) {
	fireCount := 0
	nextFire := next(time.Now())
	logger.Debugf("schedule %s: next firing at %s", name, nextFire.Format(time.RFC3339))
	timer := time.NewTimer(time.Until(nextFire))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			fireCount++
			if limit > 0 {
				logger.Debugf("schedule %s: firing (run %d of %d)", name, fireCount, limit)
			} else {
				logger.Debugf("schedule %s: firing (run %d)", name, fireCount)
			}
			fire()
			if limit > 0 && fireCount >= limit {
				logger.Infof("schedule %s: reached limit of %d run(s) - schedule stopped", name, limit)
				return
			}
			nextFire = next(time.Now())
			logger.Debugf("schedule %s: next firing at %s", name, nextFire.Format(time.RFC3339))
			timer.Reset(time.Until(nextFire))
		case <-ctx.Done():
			logger.Debugf("schedule %s: stopped after %d run(s)", name, fireCount)
			return
		}
	}
}
