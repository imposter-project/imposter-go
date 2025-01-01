package store

import (
	"os"
	"testing"
	"time"
)

func setupRedisTest(t *testing.T) *RedisStoreProvider {
	// Skip if Redis is not available
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("Skipping Redis tests: REDIS_ADDR not set")
	}

	provider := &RedisStoreProvider{}
	provider.InitStores()

	// Clear test data
	provider.client.FlushDB(provider.ctx)

	return provider
}

func TestRedisStore(t *testing.T) {
	provider := setupRedisTest(t)

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
		provider.StoreValue("test", "prefix.key1", "value1")
		provider.StoreValue("test", "prefix.key2", "value2")
		provider.StoreValue("test", "other.key3", "value3")

		values := provider.GetAllValues("test", "prefix")
		if len(values) != 2 {
			t.Errorf("Expected 2 values but got %d", len(values))
		}
		if values["key1"] != "value1" || values["key2"] != "value2" {
			t.Error("Got unexpected values")
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

	t.Run("Expiration", func(t *testing.T) {
		// Set a short expiration for testing
		os.Setenv("IMPOSTER_STORE_REDIS_EXPIRY", "1s")
		defer os.Unsetenv("IMPOSTER_STORE_REDIS_EXPIRY")

		provider.StoreValue("test", "expiring", "value")

		// Wait for expiration
		time.Sleep(2 * time.Second)

		_, found := provider.GetValue("test", "expiring")
		if found {
			t.Error("Value should have expired")
		}
	})
}

func TestRedisConnection(t *testing.T) {
	t.Run("InvalidConnection", func(t *testing.T) {
		os.Setenv("REDIS_ADDR", "localhost:1") // Invalid port
		defer os.Unsetenv("REDIS_ADDR")

		provider := &RedisStoreProvider{}
		provider.InitStores()

		// Operations should fail gracefully
		provider.StoreValue("test", "key", "value")
		_, found := provider.GetValue("test", "key")
		if found {
			t.Error("Expected operation to fail with invalid connection")
		}
	})
}
