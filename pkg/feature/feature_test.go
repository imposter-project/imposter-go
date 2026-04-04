package feature

import (
	"sync"
	"testing"
)

// Each test registers flags under a unique name so the global registry
// doesn't collide across cases.
func newFlag(t *testing.T, name, envVar string, def bool) *Flag {
	t.Helper()
	// Guard against stale registry state if a previous test panicked mid-run.
	mu.Lock()
	delete(registry, name)
	mu.Unlock()
	Reset()
	return Register(Flag{Name: name, EnvVar: envVar, Default: def})
}

func TestBool_DefaultWhenUnset(t *testing.T) {
	f := newFlag(t, "test.defaultTrue", "IMPOSTER_TEST_DEFAULT_TRUE", true)
	t.Setenv("IMPOSTER_TEST_DEFAULT_TRUE", "")
	// t.Setenv sets the var to empty; LookupEnv still reports it as set,
	// so the parsing path treats empty as "unrecognised" and falls back
	// to the default. That is the behaviour we want to lock in.
	if got := Bool(f); got != true {
		t.Fatalf("expected default true, got %v", got)
	}

	g := newFlag(t, "test.defaultFalse", "IMPOSTER_TEST_DEFAULT_FALSE", false)
	// Explicitly do not set the env var.
	if got := Bool(g); got != false {
		t.Fatalf("expected default false, got %v", got)
	}
}

func TestBool_OverrideTrue(t *testing.T) {
	f := newFlag(t, "test.overrideTrue", "IMPOSTER_TEST_OVERRIDE_TRUE", false)
	t.Setenv("IMPOSTER_TEST_OVERRIDE_TRUE", "true")
	if got := Bool(f); got != true {
		t.Fatalf("expected true, got %v", got)
	}
}

func TestBool_OverrideTrueMixedCase(t *testing.T) {
	f := newFlag(t, "test.overrideTrueMixed", "IMPOSTER_TEST_OVERRIDE_TRUE_MIXED", false)
	t.Setenv("IMPOSTER_TEST_OVERRIDE_TRUE_MIXED", "TRUE")
	if got := Bool(f); got != true {
		t.Fatalf("expected true from mixed case, got %v", got)
	}
}

func TestBool_OverrideFalse(t *testing.T) {
	f := newFlag(t, "test.overrideFalse", "IMPOSTER_TEST_OVERRIDE_FALSE", true)
	t.Setenv("IMPOSTER_TEST_OVERRIDE_FALSE", "false")
	if got := Bool(f); got != false {
		t.Fatalf("expected false, got %v", got)
	}
}

func TestBool_UnrecognisedValueFallsBackToDefault(t *testing.T) {
	f := newFlag(t, "test.unrecognised", "IMPOSTER_TEST_UNRECOGNISED", true)
	t.Setenv("IMPOSTER_TEST_UNRECOGNISED", "yes")
	if got := Bool(f); got != true {
		t.Fatalf("expected default true for unrecognised value, got %v", got)
	}
}

func TestBool_CacheSticky(t *testing.T) {
	f := newFlag(t, "test.cache", "IMPOSTER_TEST_CACHE", false)
	t.Setenv("IMPOSTER_TEST_CACHE", "true")
	if got := Bool(f); got != true {
		t.Fatalf("initial read expected true, got %v", got)
	}
	// Mutating the env after first read must not affect subsequent reads.
	t.Setenv("IMPOSTER_TEST_CACHE", "false")
	if got := Bool(f); got != true {
		t.Fatalf("cached read expected true, got %v", got)
	}
}

func TestReset_ClearsCache(t *testing.T) {
	f := newFlag(t, "test.reset", "IMPOSTER_TEST_RESET", false)
	t.Setenv("IMPOSTER_TEST_RESET", "true")
	_ = Bool(f)
	t.Setenv("IMPOSTER_TEST_RESET", "false")
	Reset()
	if got := Bool(f); got != false {
		t.Fatalf("after Reset expected fresh read false, got %v", got)
	}
}

func TestRegister_DuplicatePanics(t *testing.T) {
	_ = newFlag(t, "test.duplicate", "IMPOSTER_TEST_DUPLICATE", false)
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on duplicate Register")
		}
	}()
	Register(Flag{Name: "test.duplicate", EnvVar: "IMPOSTER_TEST_DUPLICATE_2", Default: true})
}

func TestRegister_EmptyNamePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on empty Name")
		}
	}()
	Register(Flag{EnvVar: "IMPOSTER_TEST_EMPTY_NAME"})
}

func TestRegister_EmptyEnvVarPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on empty EnvVar")
		}
	}()
	Register(Flag{Name: "test.emptyEnvVar"})
}

func TestAll_SortedByName(t *testing.T) {
	_ = newFlag(t, "test.all.b", "IMPOSTER_TEST_ALL_B", false)
	_ = newFlag(t, "test.all.a", "IMPOSTER_TEST_ALL_A", false)
	_ = newFlag(t, "test.all.c", "IMPOSTER_TEST_ALL_C", false)
	all := All()
	// Collect just our three test entries and verify relative ordering.
	var seen []string
	for _, f := range all {
		if f.Name == "test.all.a" || f.Name == "test.all.b" || f.Name == "test.all.c" {
			seen = append(seen, f.Name)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 test.all.* entries, got %v", seen)
	}
	if !(seen[0] == "test.all.a" && seen[1] == "test.all.b" && seen[2] == "test.all.c") {
		t.Fatalf("expected sorted order, got %v", seen)
	}
}

func TestBool_ConcurrentFirstReadIsConsistent(t *testing.T) {
	f := newFlag(t, "test.concurrent", "IMPOSTER_TEST_CONCURRENT", false)
	t.Setenv("IMPOSTER_TEST_CONCURRENT", "true")
	var wg sync.WaitGroup
	results := make([]bool, 32)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = Bool(f)
		}(i)
	}
	wg.Wait()
	for i, r := range results {
		if !r {
			t.Fatalf("goroutine %d saw false, expected true", i)
		}
	}
}
