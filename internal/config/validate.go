package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/robfig/cron/v3"
)

// Validate checks the static integrity of a loaded config. It returns an
// error for conditions that should prevent startup, and logs warnings for
// recoverable issues.
func Validate(cfg *Config) error {
	for name, upstream := range cfg.Upstreams {
		u, err := url.Parse(upstream.URL)
		if err != nil {
			return fmt.Errorf("upstream %q has an invalid URL %q: %w", name, upstream.URL, err)
		}
		if u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("upstream %q URL %q must include a scheme and host", name, upstream.URL)
		}
	}

	for i := range cfg.Resources {
		res := &cfg.Resources[i]

		if res.Response != nil && len(res.Responses) > 0 {
			return fmt.Errorf("resource (path %q) declares both 'response' and 'responses'; use one or the other", res.Path)
		}

		if err := validateWebSocketFields(cfg, res); err != nil {
			return err
		}

		if res.Passthrough == "" {
			continue
		}
		if _, ok := cfg.Upstreams[res.Passthrough]; !ok {
			return fmt.Errorf("resource (path %q) references unknown upstream %q", res.Path, res.Passthrough)
		}
		if res.Response != nil || len(res.Responses) > 0 || len(res.Steps) > 0 {
			logger.Warnf("resource (path %q) declares passthrough %q alongside a response or steps; passthrough takes precedence", res.Path, res.Passthrough)
		}
	}

	for i := range cfg.Interceptors {
		interceptor := &cfg.Interceptors[i]
		if interceptor.Passthrough != "" {
			logger.Warnf("interceptor (path %q) declares passthrough %q, which is not supported and will be ignored", interceptor.Path, interceptor.Passthrough)
		}
		if len(interceptor.Responses) > 0 {
			return fmt.Errorf("interceptor (path %q) declares 'responses', which is not supported for interceptors; use 'response'", interceptor.Path)
		}
		if len(interceptor.Schedule) > 0 {
			return fmt.Errorf("interceptor (path %q) declares 'schedule', which is not supported for interceptors", interceptor.Path)
		}
	}

	for i := range cfg.Schedules {
		sched := &cfg.Schedules[i]
		if err := validateScheduleTrigger(sched, scheduleDesc(sched, i)); err != nil {
			return err
		}
		if len(sched.Steps) == 0 {
			return fmt.Errorf("%s must declare at least one step", scheduleDesc(sched, i))
		}
		if sched.Response != nil || len(sched.Responses) > 0 {
			return fmt.Errorf("%s declares a response, which is only supported for schedules on websocket resources", scheduleDesc(sched, i))
		}
	}

	return nil
}

// validateWebSocketFields checks the websocket-specific resource fields
// ('on', 'responses' and 'schedule') according to the config's plugin.
func validateWebSocketFields(cfg *Config, res *Resource) error {
	if cfg.Plugin != "websocket" {
		if res.On != "" {
			return fmt.Errorf("resource (path %q) declares 'on', which is only supported by the websocket plugin", res.Path)
		}
		if len(res.Responses) > 0 {
			return fmt.Errorf("resource (path %q) declares 'responses', which is only supported by the websocket plugin", res.Path)
		}
		if len(res.Schedule) > 0 {
			return fmt.Errorf("resource (path %q) declares 'schedule', which is only supported by the websocket plugin", res.Path)
		}
		return nil
	}

	switch res.On {
	case "", WebSocketEventOpen, WebSocketEventMessage, WebSocketEventClose:
	default:
		return fmt.Errorf("resource (path %q) has invalid 'on' value %q; must be one of open, message, close", res.Path, res.On)
	}

	if len(res.Schedule) > 0 && res.NormalisedOn() != WebSocketEventOpen {
		return fmt.Errorf("resource (path %q) declares 'schedule' but is not an 'on: open' resource; connection-scoped schedules must be declared on the open resource", res.Path)
	}

	if res.NormalisedOn() == WebSocketEventClose && (res.Response != nil || len(res.Responses) > 0) {
		logger.Warnf("resource (path %q) declares a response on an 'on: close' resource; no message can be sent after the connection has closed", res.Path)
	}

	if res.Passthrough != "" {
		return fmt.Errorf("resource (path %q) declares 'passthrough', which is not supported by the websocket plugin", res.Path)
	}

	if res.Method != "" {
		logger.Warnf("resource (path %q) declares 'method', which has no effect for websocket resources", res.Path)
	}

	for i := range res.Schedule {
		sched := &res.Schedule[i]
		desc := scheduleDesc(sched, i)
		if err := validateScheduleTrigger(sched, desc); err != nil {
			return err
		}
		if sched.Response != nil && len(sched.Responses) > 0 {
			return fmt.Errorf("%s declares both 'response' and 'responses'; use one or the other", desc)
		}
		if sched.Response == nil && len(sched.Responses) == 0 && len(sched.Steps) == 0 {
			return fmt.Errorf("%s must declare a response or at least one step", desc)
		}
	}

	return nil
}

// validateScheduleTrigger ensures a schedule entry declares exactly one of
// 'every' or 'cron', and that the value parses.
func validateScheduleTrigger(sched *Schedule, desc string) error {
	if (sched.Every == "") == (sched.Cron == "") {
		return fmt.Errorf("%s must declare exactly one of 'every' or 'cron'", desc)
	}
	if sched.Every != "" {
		d, err := time.ParseDuration(sched.Every)
		if err != nil {
			return fmt.Errorf("%s has invalid 'every' duration %q: %w", desc, sched.Every, err)
		}
		if d <= 0 {
			return fmt.Errorf("%s 'every' duration must be positive, got %q", desc, sched.Every)
		}
	}
	if sched.Cron != "" {
		if _, err := cron.ParseStandard(sched.Cron); err != nil {
			return fmt.Errorf("%s has invalid 'cron' expression %q: %w", desc, sched.Cron, err)
		}
	}
	return nil
}

func scheduleDesc(sched *Schedule, idx int) string {
	if sched.Name != "" {
		return fmt.Sprintf("schedule %q", sched.Name)
	}
	return fmt.Sprintf("schedule at index %d", idx)
}
