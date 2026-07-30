package store

import (
	"os"
	"testing"
	"time"
)

// TestInMemoryStore_AtomicIncrement exercises the atomic counter behaviour of
// the in-memory provider, including the type coercion applied to any existing
// value before incrementing.
func TestInMemoryStore_AtomicIncrement(t *testing.T) {
	t.Run("IncrementNewKeyStartsFromZero", func(t *testing.T) {
		provider := setupInMemoryTest(t)
		val, err := provider.AtomicIncrement("counters", "new", 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != 5 {
			t.Errorf("expected 5 but got %d", val)
		}
	})

	t.Run("SuccessiveIncrementsAccumulate", func(t *testing.T) {
		provider := setupInMemoryTest(t)
		if _, err := provider.AtomicIncrement("counters", "hits", 1); err != nil {
			t.Fatal(err)
		}
		val, err := provider.AtomicIncrement("counters", "hits", 2)
		if err != nil {
			t.Fatal(err)
		}
		if val != 3 {
			t.Errorf("expected 3 but got %d", val)
		}
	})

	// The stored value may originate from JSON (float64), a script (int) or a
	// previous atomic operation (int64); all should be coerced to int64.
	t.Run("CoercesExistingValueTypes", func(t *testing.T) {
		cases := []struct {
			name    string
			initial interface{}
			want    int64
		}{
			{name: "int64", initial: int64(10), want: 11},
			{name: "int", initial: 7, want: 8},
			{name: "float64", initial: float64(41), want: 42},
			{name: "non-numeric string treated as zero", initial: "not-a-number", want: 1},
			{name: "bool treated as zero", initial: true, want: 1},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				provider := setupInMemoryTest(t)
				provider.StoreValue("counters", "seed", tc.initial)
				val, err := provider.AtomicIncrement("counters", "seed", 1)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if val != tc.want {
					t.Errorf("expected %d but got %d", tc.want, val)
				}
			})
		}
	})

	t.Run("StoredValueIsReadableAfterIncrement", func(t *testing.T) {
		provider := setupInMemoryTest(t)
		if _, err := provider.AtomicIncrement("counters", "shared", 4); err != nil {
			t.Fatal(err)
		}
		val, found := provider.GetValue("counters", "shared")
		if !found {
			t.Fatal("expected the incremented value to be retrievable")
		}
		if val != int64(4) {
			t.Errorf("expected int64(4) but got %v (%T)", val, val)
		}
	})
}

// TestInMemoryStore_AtomicDecrement verifies decrement is the inverse of
// increment, including crossing below zero.
func TestInMemoryStore_AtomicDecrement(t *testing.T) {
	provider := setupInMemoryTest(t)

	t.Run("DecrementNewKeyGoesNegative", func(t *testing.T) {
		val, err := provider.AtomicDecrement("counters", "credits", 3)
		if err != nil {
			t.Fatal(err)
		}
		if val != -3 {
			t.Errorf("expected -3 but got %d", val)
		}
	})

	t.Run("DecrementExistingValue", func(t *testing.T) {
		provider.StoreValue("counters", "stock", int64(10))
		val, err := provider.AtomicDecrement("counters", "stock", 4)
		if err != nil {
			t.Fatal(err)
		}
		if val != 6 {
			t.Errorf("expected 6 but got %d", val)
		}
	})
}

// TestInMemoryStore_AtomicIncrementTTL confirms that a TTL is applied when a
// counter is first created and that the counter expires like any other value.
func TestInMemoryStore_AtomicIncrementTTL(t *testing.T) {
	oldTTL := os.Getenv("IMPOSTER_STORE_INMEMORY_TTL")
	defer func() {
		if oldTTL == "" {
			os.Unsetenv("IMPOSTER_STORE_INMEMORY_TTL")
		} else {
			os.Setenv("IMPOSTER_STORE_INMEMORY_TTL", oldTTL)
		}
	}()

	os.Setenv("IMPOSTER_STORE_INMEMORY_TTL", "1")
	provider := setupInMemoryTest(t)

	if _, err := provider.AtomicIncrement("counters", "ephemeral", 1); err != nil {
		t.Fatal(err)
	}

	// It should be present immediately.
	if _, found := provider.GetValue("counters", "ephemeral"); !found {
		t.Fatal("expected counter to exist immediately after increment")
	}

	// It should expire within a reasonable window (buffered for slow CI).
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, found := provider.GetValue("counters", "ephemeral"); !found {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Error("expected counter to expire after its TTL")
}

// TestRequestStoreProvider_Atomic verifies the request-scoped provider delegates
// atomic operations to its backing in-memory store.
func TestRequestStoreProvider_Atomic(t *testing.T) {
	provider := setupRequestTest(t)

	inc, err := provider.AtomicIncrement("request", "count", 2)
	if err != nil {
		t.Fatal(err)
	}
	if inc != 2 {
		t.Errorf("expected 2 but got %d", inc)
	}

	dec, err := provider.AtomicDecrement("request", "count", 1)
	if err != nil {
		t.Fatal(err)
	}
	if dec != 1 {
		t.Errorf("expected 1 but got %d", dec)
	}
}
