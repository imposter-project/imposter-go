// Package feature provides a small, general-purpose registry for
// boolean runtime feature flags.
//
// Flags are declared once, from the init() of the package that owns them,
// via Register. Callers hold on to the returned *Flag and read its
// current value with Bool, which resolves the backing environment variable
// exactly once per flag and caches the result.
//
// The parsing convention matches the long-standing idiom in this codebase:
// an env var set to "true" (case-insensitive) is true, "false" is false,
// anything else - including unset - falls back to the declared default.
package feature

import (
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/imposter-project/imposter-go/pkg/logger"
)

// Flag is a declarative description of a runtime boolean toggle.
type Flag struct {
	// Name is a stable dotted identifier, e.g. "config.scanRecursive".
	// Used as the registry key and for logging / discovery.
	Name string
	// EnvVar is the environment variable consulted at read time.
	EnvVar string
	// Default is returned when the env var is unset or unrecognised.
	Default bool
	// Description is a short human-readable explanation shown by All().
	Description string
}

var (
	mu       sync.RWMutex
	registry = map[string]*Flag{}
	cache    sync.Map // flag name -> bool
)

// Register declares a flag. It must be called before any call to Bool for
// the same flag - typically from a package-level var initialiser so
// registration runs during package init.
//
// Registering two flags with the same Name panics, which surfaces
// copy/paste bugs at startup rather than letting them lurk.
func Register(f Flag) *Flag {
	if f.Name == "" {
		panic("feature: Register called with empty Name")
	}
	if f.EnvVar == "" {
		panic("feature: Register called with empty EnvVar for flag " + f.Name)
	}
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[f.Name]; exists {
		panic("feature: flag already registered: " + f.Name)
	}
	stored := f
	registry[f.Name] = &stored
	return &stored
}

// Bool returns the effective value of a previously-registered flag. The
// first call for a given flag reads the environment and logs the
// resolution at Trace level; subsequent calls are served from cache.
func Bool(f *Flag) bool {
	if f == nil {
		return false
	}
	if v, ok := cache.Load(f.Name); ok {
		return v.(bool)
	}
	raw, set := os.LookupEnv(f.EnvVar)
	value := f.Default
	source := "default"
	if set {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "true":
			value = true
			source = "from " + f.EnvVar
		case "false":
			value = false
			source = "from " + f.EnvVar
		default:
			// Unrecognised value - keep the default but note the source.
			source = "default (ignored unrecognised " + f.EnvVar + "=" + raw + ")"
		}
	}
	// LoadOrStore ensures a single stable value even under races.
	actual, loaded := cache.LoadOrStore(f.Name, value)
	if !loaded {
		logger.Tracef("feature flag %s = %t (%s)", f.Name, value, source)
	}
	return actual.(bool)
}

// Reset clears the value cache. It is intended for tests that mutate the
// environment between cases; production code should never call it.
func Reset() {
	cache.Range(func(k, _ any) bool {
		cache.Delete(k)
		return true
	})
}

// All returns a snapshot of every registered flag, sorted by Name. Useful
// for startup banners or a "list feature flags" diagnostic endpoint.
func All() []Flag {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Flag, 0, len(registry))
	for _, f := range registry {
		out = append(out, *f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
