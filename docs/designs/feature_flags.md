# Feature flags

`pkg/feature` is a small, dependency-free registry for boolean runtime
toggles. It exists so that the codebase has one consistent way to declare,
resolve and discover feature flags.

This document is aimed at developers working on imposter-go. For the
user-facing list of environment variables, see [env_vars.md](env_vars.md).

## What belongs in the registry

The registry is intentionally narrow in scope:

- **In scope:** boolean toggles that turn a behaviour on or off at runtime
  (e.g. "scan the config directory recursively", "enable legacy config
  support").
- **Out of scope:** string or numeric configuration values - log level,
  store driver, TTLs, config dir, port, and similar. These are
  configuration, not feature flags, and continue to be read directly from
  the environment or `ImposterConfig`.

If you find yourself parsing a value beyond `true` / `false`, it does not
belong here.

## Declaring a flag

Flags are declared at package level so that registration runs during
package `init()`. Keep the declaration close to the code that reads it.

```go
package config

import "github.com/imposter-project/imposter-go/pkg/feature"

var flagConfigScanRecursive = feature.Register(feature.Flag{
    Name:        "config.scanRecursive",
    EnvVar:      "IMPOSTER_CONFIG_SCAN_RECURSIVE",
    Default:     false,
    Description: "Scan the config directory recursively for config files.",
})
```

Field conventions:

- **`Name`** - stable dotted identifier, `<package>.<camelCaseFlag>`. This
  is the registry key, so it must be unique; registering the same name
  twice panics at startup (deliberately, to surface copy/paste bugs).
- **`EnvVar`** - the `IMPOSTER_*` environment variable that backs the
  flag. Preserve existing names when migrating, even if they do not match
  the `Name` exactly.
- **`Default`** - the value returned when the env var is unset or holds an
  unrecognised value. Default to the safer / backward-compatible option.
- **`Description`** - one short sentence. Surfaced by `feature.All()` and
  is the main thing a future reader will see when auditing flags.

## Reading a flag

```go
if feature.Bool(flagConfigScanRecursive) {
    // recursive walk
}
```

`feature.Bool` resolves the env var on first call, caches the result for
the lifetime of the process, and logs the resolution once at `Trace` level
(including whether the value came from the env var or the default). This
means:

- Subsequent reads are effectively free - call `feature.Bool` directly at
  the point of use rather than caching the result in a local variable.
- Changing the environment after the first read has no effect. Flags are
  process-lifetime; callers should not expect live reloads.
- An unrecognised value (anything that is not `true` / `false`,
  case-insensitive, after trimming) falls back to `Default` and is logged
  so that typos are discoverable.

## Discovery

`feature.All()` returns a sorted snapshot of every registered flag. It is
intended for startup banners or a future "list feature flags" diagnostic
endpoint. It does not expose the currently-resolved value, only the
declaration, so calling it does not prime the cache.

## Testing

Tests that mutate flag-backing env vars must call `feature.Reset()` to
clear the cache between cases, otherwise a value read in one test will
leak into the next. A typical pattern:

```go
func TestSomething(t *testing.T) {
    t.Setenv("IMPOSTER_CONFIG_SCAN_RECURSIVE", "true")
    feature.Reset()
    t.Cleanup(feature.Reset)

    // ... exercise code that reads the flag
}
```

`feature.Reset()` only clears the value cache; registrations persist, so
flags declared at package init remain available.

## Current flags

| Name | Env var | Default | Purpose |
| --- | --- | --- | --- |
| `config.scanRecursive` | `IMPOSTER_CONFIG_SCAN_RECURSIVE` | `false` | Recursively walk the config directory. |
| `config.autoBasePath` | `IMPOSTER_AUTO_BASE_PATH` | `false` | Derive a resource `basePath` from each config file's relative directory. |
| `config.supportLegacy` | `IMPOSTER_SUPPORT_LEGACY_CONFIG` | `false` | Enable on-the-fly transformation of the legacy config format. |
| `external.pluginsEnabled` | `IMPOSTER_EXTERNAL_PLUGINS` | `false` | Enable loading of external (out-of-process) plugins. |

When adding a new flag, append a row here as part of the same change.
