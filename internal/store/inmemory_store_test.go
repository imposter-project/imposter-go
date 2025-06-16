package store

import (
	"os"
	"testing"
	"time"
)

func setupInMemoryTest(t *testing.T) *InMemoryStoreProvider {
	provider := &InMemoryStoreProvider{}
	provider.InitStores()
	return provider
}

func TestInMemoryStoreProvider(t *testing.T) {
	provider := setupInMemoryTest(t)

	t.Run("StoreAndGetValue", func(t *testing.T) {
		provider.StoreValue("test", "key1", "value1")
		val, found := provider.GetValue("test", "key1")
		if !found {
			t.Error("Expected to find value but got not found")
		}
		if val != "value1" {
			t.Errorf("Expected value1 but got %v", val)
		}
	})

	t.Run("GetNonExistentValue", func(t *testing.T) {
		_, found := provider.GetValue("test", "nonexistent")
		if found {
			t.Error("Expected not found for nonexistent key")
		}
	})

	t.Run("GetAllValues", func(t *testing.T) {
		// Clear any previous test data
		provider.DeleteStore("test")

		// Store some test data
		provider.StoreValue("test", "prefix.key1", "value1")
		provider.StoreValue("test", "prefix.key2", "value2")
		provider.StoreValue("test", "other.key3", "value3")

		// Get all values with prefix
		values := provider.GetAllValues("test", "prefix")

		// Debug output
		t.Logf("Stored values: %v", provider.stores["test"].data)
		t.Logf("Retrieved values: %v", values)

		// Check the number of values
		if len(values) != 2 {
			t.Errorf("Expected 2 values but got %d. Values: %v", len(values), values)
		}

		// Check each value individually
		expectedValues := map[string]string{
			"prefix.key1": "value1",
			"prefix.key2": "value2",
		}

		for key, expectedValue := range expectedValues {
			actualValue, ok := values[key]
			if !ok {
				t.Errorf("Expected to find key %q but it was missing. All values: %v", key, values)
				continue
			}
			if actualValue != expectedValue {
				t.Errorf("For key %q expected value %q but got %q", key, expectedValue, actualValue)
			}
		}

		// Verify other.key3 is not included
		if _, ok := values["key3"]; ok {
			t.Errorf("key3 should not be included in the results. All values: %v", values)
		}
	})

	t.Run("DeleteValue", func(t *testing.T) {
		provider.StoreValue("test", "key1", "value1")
		provider.DeleteValue("test", "key1")
		_, found := provider.GetValue("test", "key1")
		if found {
			t.Error("Value should have been deleted")
		}
	})

	t.Run("DeleteStore", func(t *testing.T) {
		provider.StoreValue("test", "key1", "value1")
		provider.DeleteStore("test")
		_, found := provider.GetValue("test", "key1")
		if found {
			t.Error("Store should have been deleted")
		}
	})

	t.Run("StoreComplexValue", func(t *testing.T) {
		complexValue := map[string]interface{}{
			"name": "test",
			"age":  30,
			"nested": map[string]interface{}{
				"key": "value",
			},
		}
		provider.StoreValue("test", "complex", complexValue)
		val, found := provider.GetValue("test", "complex")
		if !found {
			t.Error("Expected to find complex value")
		}
		mapVal, ok := val.(map[string]interface{})
		if !ok {
			t.Error("Expected map type for complex value")
		}
		if mapVal["name"] != "test" || mapVal["age"] != 30 {
			t.Error("Complex value not stored correctly")
		}
	})
}

func TestInMemoryStore_Concurrency(t *testing.T) {
	provider := setupInMemoryTest(t)

	t.Run("ConcurrentAccess", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				key := "key"
				value := "value"
				// Write
				provider.StoreValue("test", key, value)
				// Read
				_, _ = provider.GetValue("test", key)
				// Read all
				_ = provider.GetAllValues("test", "key")
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestInMemoryStore_TTL(t *testing.T) {
	// Set up TTL environment variable for testing
	oldTTL := os.Getenv("IMPOSTER_STORE_INMEMORY_TTL")
	defer func() {
		if oldTTL == "" {
			os.Unsetenv("IMPOSTER_STORE_INMEMORY_TTL")
		} else {
			os.Setenv("IMPOSTER_STORE_INMEMORY_TTL", oldTTL)
		}
	}()

	t.Run("TTLExpiration", func(t *testing.T) {
		// Set TTL to 1 second for testing
		os.Setenv("IMPOSTER_STORE_INMEMORY_TTL", "1")

		provider := setupInMemoryTest(t)

		// Store a value
		provider.StoreValue("test", "ttl-key", "ttl-value")

		// Verify it exists immediately
		val, found := provider.GetValue("test", "ttl-key")
		if !found {
			t.Fatal("Expected to find value immediately after storing")
		}
		if val != "ttl-value" {
			t.Errorf("Expected 'ttl-value' but got %v", val)
		}

		// Verify it exists via GetAllValues immediately
		allValues := provider.GetAllValues("test", "ttl")
		if len(allValues) != 1 {
			t.Fatalf("Expected 1 value in GetAllValues immediately, got %d", len(allValues))
		}

		// Wait for TTL + buffer for slow CI/CD systems (up to 3 seconds)
		maxWait := 3 * time.Second
		start := time.Now()

		for time.Since(start) < maxWait {
			// Check if value has expired
			_, found := provider.GetValue("test", "ttl-key")
			if !found {
				// Value expired successfully
				t.Logf("TTL expiration detected after %v", time.Since(start))
				break
			}
			// Wait a bit before checking again
			time.Sleep(100 * time.Millisecond)
		}

		// Final verification that value is expired
		_, found = provider.GetValue("test", "ttl-key")
		if found {
			t.Error("Expected value to be expired after TTL, but it still exists")
		}

		// Verify GetAllValues also doesn't return expired key
		allValues = provider.GetAllValues("test", "ttl")
		if len(allValues) != 0 {
			t.Errorf("Expected no values in GetAllValues after TTL, got %d: %v", len(allValues), allValues)
		}
	})

	t.Run("NoTTLConfigured", func(t *testing.T) {
		// Unset TTL environment variable
		os.Unsetenv("IMPOSTER_STORE_INMEMORY_TTL")

		provider := setupInMemoryTest(t)

		// Store a value
		provider.StoreValue("test", "no-ttl-key", "no-ttl-value")

		// Wait longer than the previous TTL tests
		time.Sleep(1500 * time.Millisecond)

		// Verify value still exists (no TTL configured)
		val, found := provider.GetValue("test", "no-ttl-key")
		if !found {
			t.Error("Expected value to persist when no TTL is configured")
		}
		if val != "no-ttl-value" {
			t.Errorf("Expected 'no-ttl-value' but got %v", val)
		}
	})

	t.Run("InvalidTTLConfiguration", func(t *testing.T) {
		// Set invalid TTL value
		os.Setenv("IMPOSTER_STORE_INMEMORY_TTL", "invalid")

		provider := setupInMemoryTest(t)

		// Store a value (should work despite invalid TTL)
		provider.StoreValue("test", "invalid-ttl-key", "invalid-ttl-value")

		// Wait a bit
		time.Sleep(500 * time.Millisecond)

		// Verify value still exists (invalid TTL should be ignored)
		val, found := provider.GetValue("test", "invalid-ttl-key")
		if !found {
			t.Error("Expected value to persist when TTL is invalid")
		}
		if val != "invalid-ttl-value" {
			t.Errorf("Expected 'invalid-ttl-value' but got %v", val)
		}
	})
}
