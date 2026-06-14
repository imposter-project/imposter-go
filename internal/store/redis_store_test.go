//go:build integration

package store

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func startRedisContainer(t *testing.T) *redis.RedisContainer {
	t.Helper()
	ctx := context.Background()
	container, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err, "failed to start Redis container")
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(ctx))
	})
	return container
}

func setupRedisProvider(t *testing.T, container *redis.RedisContainer) *RedisStoreProvider {
	t.Helper()
	ctx := context.Background()
	endpoint, err := container.Endpoint(ctx, "")
	require.NoError(t, err)

	t.Setenv("REDIS_ADDR", endpoint)
	t.Setenv("REDIS_PASSWORD", "")

	provider := &RedisStoreProvider{}
	provider.InitStores()
	provider.client.FlushDB(provider.ctx)
	return provider
}

func TestRedisStore_BasicCRUD(t *testing.T) {
	container := startRedisContainer(t)
	provider := setupRedisProvider(t, container)

	t.Run("StoreAndGetValue", func(t *testing.T) {
		provider.StoreValue("test", "key1", "value1")
		val, found := provider.GetValue("test", "key1")
		assert.True(t, found)
		assert.Equal(t, "value1", val)
	})

	t.Run("GetNonExistentValue", func(t *testing.T) {
		_, found := provider.GetValue("test", "nonexistent")
		assert.False(t, found)
	})

	t.Run("OverwriteValue", func(t *testing.T) {
		provider.StoreValue("test", "overwrite", "first")
		provider.StoreValue("test", "overwrite", "second")
		val, found := provider.GetValue("test", "overwrite")
		assert.True(t, found)
		assert.Equal(t, "second", val)
	})

	t.Run("DeleteValue", func(t *testing.T) {
		provider.StoreValue("test", "todelete", "value")
		provider.DeleteValue("test", "todelete")
		_, found := provider.GetValue("test", "todelete")
		assert.False(t, found)
	})

	t.Run("DeleteNonExistentValue", func(t *testing.T) {
		provider.DeleteValue("test", "never-existed")
	})

	t.Run("DeleteStore", func(t *testing.T) {
		provider.StoreValue("delstore", "k1", "v1")
		provider.StoreValue("delstore", "k2", "v2")
		provider.DeleteStore("delstore")
		_, found1 := provider.GetValue("delstore", "k1")
		_, found2 := provider.GetValue("delstore", "k2")
		assert.False(t, found1)
		assert.False(t, found2)
	})
}

func TestRedisStore_ComplexValues(t *testing.T) {
	container := startRedisContainer(t)
	provider := setupRedisProvider(t, container)

	t.Run("MapValue", func(t *testing.T) {
		v := map[string]interface{}{
			"name": "test",
			"age":  float64(30),
			"nested": map[string]interface{}{
				"key": "value",
			},
		}
		provider.StoreValue("test", "complex", v)
		val, found := provider.GetValue("test", "complex")
		require.True(t, found)
		m, ok := val.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test", m["name"])
		assert.Equal(t, float64(30), m["age"])
		nested, ok := m["nested"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", nested["key"])
	})

	t.Run("ArrayValue", func(t *testing.T) {
		v := []interface{}{"a", "b", "c"}
		provider.StoreValue("test", "arr", v)
		val, found := provider.GetValue("test", "arr")
		require.True(t, found)
		arr, ok := val.([]interface{})
		require.True(t, ok)
		assert.Equal(t, []interface{}{"a", "b", "c"}, arr)
	})

	t.Run("NumericValue", func(t *testing.T) {
		provider.StoreValue("test", "num", 42.5)
		val, found := provider.GetValue("test", "num")
		require.True(t, found)
		assert.Equal(t, 42.5, val)
	})

	t.Run("BooleanValue", func(t *testing.T) {
		provider.StoreValue("test", "flag", true)
		val, found := provider.GetValue("test", "flag")
		require.True(t, found)
		assert.Equal(t, true, val)
	})

	t.Run("NullValue", func(t *testing.T) {
		provider.StoreValue("test", "null", nil)
		val, found := provider.GetValue("test", "null")
		require.True(t, found)
		assert.Nil(t, val)
	})
}

func TestRedisStore_GetAllValues(t *testing.T) {
	container := startRedisContainer(t)
	provider := setupRedisProvider(t, container)

	provider.StoreValue("test", "prefix.key1", "value1")
	provider.StoreValue("test", "prefix.key2", "value2")
	provider.StoreValue("test", "other.key3", "value3")

	t.Run("WithMatchingPrefix", func(t *testing.T) {
		values := provider.GetAllValues("test", "prefix")
		assert.Len(t, values, 2)
		assert.Equal(t, "value1", values["prefix.key1"])
		assert.Equal(t, "value2", values["prefix.key2"])
	})

	t.Run("WithNoMatchingPrefix", func(t *testing.T) {
		values := provider.GetAllValues("test", "nomatch")
		assert.Empty(t, values)
	})

	t.Run("EmptyPrefix", func(t *testing.T) {
		values := provider.GetAllValues("test", "")
		assert.Len(t, values, 3)
	})

	t.Run("AcrossStores", func(t *testing.T) {
		provider.StoreValue("store-a", "shared.k", "from-a")
		provider.StoreValue("store-b", "shared.k", "from-b")
		valA := provider.GetAllValues("store-a", "shared")
		valB := provider.GetAllValues("store-b", "shared")
		assert.Equal(t, "from-a", valA["shared.k"])
		assert.Equal(t, "from-b", valB["shared.k"])
	})
}

func TestRedisStore_Expiration(t *testing.T) {
	container := startRedisContainer(t)
	provider := setupRedisProvider(t, container)

	t.Setenv("IMPOSTER_STORE_REDIS_EXPIRY", "1s")

	provider.StoreValue("test", "expiring", "value")
	val, found := provider.GetValue("test", "expiring")
	require.True(t, found)
	assert.Equal(t, "value", val)

	time.Sleep(2 * time.Second)

	_, found = provider.GetValue("test", "expiring")
	assert.False(t, found, "value should have expired")
}

func TestRedisStore_AtomicOperations(t *testing.T) {
	container := startRedisContainer(t)
	provider := setupRedisProvider(t, container)

	t.Run("Increment", func(t *testing.T) {
		val, err := provider.AtomicIncrement("test", "counter", 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)

		val, err = provider.AtomicIncrement("test", "counter", 5)
		require.NoError(t, err)
		assert.Equal(t, int64(6), val)
	})

	t.Run("Decrement", func(t *testing.T) {
		provider.client.FlushDB(provider.ctx)

		_, err := provider.AtomicIncrement("test", "counter", 10)
		require.NoError(t, err)

		val, err := provider.AtomicDecrement("test", "counter", 3)
		require.NoError(t, err)
		assert.Equal(t, int64(7), val)
	})

	t.Run("ConcurrentIncrements", func(t *testing.T) {
		provider.client.FlushDB(provider.ctx)

		const goroutines = 50
		const incrementsPerGoroutine = 20
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < incrementsPerGoroutine; j++ {
					_, err := provider.AtomicIncrement("test", "concurrent", 1)
					assert.NoError(t, err)
				}
			}()
		}
		wg.Wait()

		val, found := provider.GetValue("test", "concurrent")
		require.True(t, found)
		expected := fmt.Sprintf("%d", goroutines*incrementsPerGoroutine)
		assert.Equal(t, expected, fmt.Sprintf("%v", val))
	})
}

func TestRedisStore_KeyPrefix(t *testing.T) {
	container := startRedisContainer(t)

	t.Setenv("IMPOSTER_STORE_KEY_PREFIX", "myprefix")
	provider := setupRedisProvider(t, container)

	provider.StoreValue("test", "key1", "value1")
	val, found := provider.GetValue("test", "key1")
	assert.True(t, found)
	assert.Equal(t, "value1", val)

	rawVal := provider.client.Get(provider.ctx, "test:myprefix.key1").Val()
	assert.NotEmpty(t, rawVal, "key should be stored with prefix in Redis")

	os.Unsetenv("IMPOSTER_STORE_KEY_PREFIX")
}

func TestRedisStore_StoreIsolation(t *testing.T) {
	container := startRedisContainer(t)
	provider := setupRedisProvider(t, container)

	provider.StoreValue("store1", "key", "value-from-store1")
	provider.StoreValue("store2", "key", "value-from-store2")

	val1, found := provider.GetValue("store1", "key")
	assert.True(t, found)
	assert.Equal(t, "value-from-store1", val1)

	val2, found := provider.GetValue("store2", "key")
	assert.True(t, found)
	assert.Equal(t, "value-from-store2", val2)

	provider.DeleteStore("store1")
	_, found = provider.GetValue("store1", "key")
	assert.False(t, found, "store1 should be deleted")

	val2, found = provider.GetValue("store2", "key")
	assert.True(t, found, "store2 should be unaffected")
	assert.Equal(t, "value-from-store2", val2)
}

func TestRedisStore_InvalidConnection(t *testing.T) {
	t.Setenv("REDIS_ADDR", "localhost:1")

	provider := &RedisStoreProvider{}
	provider.InitStores()

	provider.StoreValue("test", "key", "value")
	_, found := provider.GetValue("test", "key")
	assert.False(t, found, "expected operation to fail with invalid connection")
}
