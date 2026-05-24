package config

import (
	"fmt"
	"net/url"

	"github.com/imposter-project/imposter-go/pkg/logger"
)

// Validate checks the static integrity of a loaded config, focusing on the
// upstream/passthrough configuration. It returns an error for conditions that
// should prevent startup, and logs warnings for recoverable issues.
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
		if res.Passthrough == "" {
			continue
		}
		if _, ok := cfg.Upstreams[res.Passthrough]; !ok {
			return fmt.Errorf("resource (path %q) references unknown upstream %q", res.Path, res.Passthrough)
		}
		if res.Response != nil || len(res.Steps) > 0 {
			logger.Warnf("resource (path %q) declares passthrough %q alongside a response or steps; passthrough takes precedence", res.Path, res.Passthrough)
		}
	}

	for i := range cfg.Interceptors {
		if cfg.Interceptors[i].Passthrough != "" {
			logger.Warnf("interceptor (path %q) declares passthrough %q, which is not supported and will be ignored", cfg.Interceptors[i].Path, cfg.Interceptors[i].Passthrough)
		}
	}

	return nil
}
