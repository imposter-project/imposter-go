package ratelimiter

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
)

// setupDynamoDBIntegrationTest sets up DynamoDB store for integration testing
func setupDynamoDBIntegrationTest(t *testing.T) store.StoreProvider {
	// Skip if DynamoDB configuration is not available
	tableName := os.Getenv("IMPOSTER_DYNAMODB_TABLE")
	if tableName == "" {
		t.Skip("Skipping DynamoDB integration tests: IMPOSTER_DYNAMODB_TABLE not set")
	}

	// Set environment to use DynamoDB
	oldDriver := os.Getenv("IMPOSTER_STORE_DRIVER")
	os.Setenv("IMPOSTER_STORE_DRIVER", "store-dynamodb")

	// Initialize the global store provider
	store.InitStoreProvider()

	// Restore the environment variable after test
	t.Cleanup(func() {
		if oldDriver == "" {
			os.Unsetenv("IMPOSTER_STORE_DRIVER")
		} else {
			os.Setenv("IMPOSTER_STORE_DRIVER", oldDriver)
		}
	})

	return store.GetStoreProvider()
}

// setupRedisIntegrationTest sets up Redis store for integration testing
func setupRedisIntegrationTest(t *testing.T) store.StoreProvider {
	// Skip if Redis is not available
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		t.Skip("Skipping Redis integration tests: REDIS_ADDR not set")
	}

	// Set environment to use Redis
	oldDriver := os.Getenv("IMPOSTER_STORE_DRIVER")
	os.Setenv("IMPOSTER_STORE_DRIVER", "store-redis")

	// Initialize the global store provider
	store.InitStoreProvider()

	// Restore the environment variable after test
	t.Cleanup(func() {
		if oldDriver == "" {
			os.Unsetenv("IMPOSTER_STORE_DRIVER")
		} else {
			os.Setenv("IMPOSTER_STORE_DRIVER", oldDriver)
		}
	})

	storeProvider := store.GetStoreProvider()

	// Clear any existing test data
	if redisProvider, ok := storeProvider.(*store.RedisStoreProvider); ok {
		// Access to clear method would require exposing it or using a different approach
		// For now, we'll rely on TTL to clean up old data
		_ = redisProvider
	}

	return storeProvider
}

// runRateLimiterIntegrationTest runs the standard rate limiter test suite with any store backend
func runRateLimiterIntegrationTest(t *testing.T, storeProvider store.StoreProvider, testName string) {
	t.Run(fmt.Sprintf("%s_BasicRateLimiting", testName), func(t *testing.T) {
		rl := NewRateLimiter(storeProvider)
		defer rl.(*RateLimiterImpl).Stop()

		resourceKey := "GET:/test"
		limits := []config.ConcurrencyLimit{
			{
				Limit: 3,
				Response: &config.Response{
					StatusCode: 429,
					Content:    "Rate limited",
				},
			},
		}

		// Test requests within limit (up to 3)
		for i := 0; i < 3; i++ {
			result, err := rl.CheckAndIncrement(resourceKey, limits)
			if err != nil {
				t.Fatalf("unexpected error on request %d: %v", i, err)
			}
			if result != nil {
				t.Fatalf("expected no rate limit on request %d, got: %+v", i, result)
			}
		}

		// Test request that exceeds limit (4th request)
		result, err := rl.CheckAndIncrement(resourceKey, limits)
		if err != nil {
			t.Fatalf("unexpected error on excess request: %v", err)
		}
		if result == nil {
			t.Fatal("expected rate limit response, got nil")
		}
		if result.Response.StatusCode != 429 {
			t.Fatalf("expected status code 429, got %d", result.Response.StatusCode)
		}

		// Test decrement allows new request
		err = rl.Decrement(resourceKey)
		if err != nil {
			t.Fatalf("unexpected error on decrement: %v", err)
		}

		result, err = rl.CheckAndIncrement(resourceKey, limits)
		if err != nil {
			t.Fatalf("unexpected error on new request after decrement: %v", err)
		}
		if result != nil {
			t.Fatalf("expected no rate limit after decrement, got: %+v", result)
		}
	})

	t.Run(fmt.Sprintf("%s_MultipleInstances", testName), func(t *testing.T) {
		rl := NewRateLimiter(storeProvider)
		defer rl.(*RateLimiterImpl).Stop()

		resourceKey := "GET:/multi-instance"
		limits := []config.ConcurrencyLimit{
			{
				Limit: 5,
				Response: &config.Response{
					StatusCode: 503,
					Content:    "Service overloaded",
				},
			},
		}

		// Simulate multiple server instances
		instanceIds := []string{"server-1", "server-2", "server-3"}
		requestsPerInstance := 2 // Total: 6 requests, should exceed limit of 5

		var rateLimitedCount int

		for _, instanceID := range instanceIds {
			for j := 0; j < requestsPerInstance; j++ {
				result, err := rl.CheckAndIncrement(resourceKey, limits)
				if err != nil {
					t.Fatalf("unexpected error for instance %s request %d: %v", instanceID, j, err)
				}
				if result != nil {
					rateLimitedCount++
				}
			}
		}

		// Should have at least one rate limited request since we exceeded the limit
		if rateLimitedCount == 0 {
			t.Error("expected some requests to be rate limited, but none were")
		}

		t.Logf("Rate limited %d out of %d total requests across multiple instances",
			rateLimitedCount, len(instanceIds)*requestsPerInstance)
	})

	t.Run(fmt.Sprintf("%s_TTLBehavior", testName), func(t *testing.T) {
		// Use shorter TTL for integration testing
		rl := NewRateLimiterWithTTL(storeProvider, 2*time.Second)
		defer rl.(*RateLimiterImpl).Stop()

		resourceKey := "GET:/ttl-test"
		limits := []config.ConcurrencyLimit{
			{
				Limit: 2,
				Response: &config.Response{
					StatusCode: 429,
					Content:    "Rate limited",
				},
			},
		}

		// Fill up to the limit
		for i := 0; i < 2; i++ {
			result, err := rl.CheckAndIncrement(resourceKey, limits)
			if err != nil {
				t.Fatalf("unexpected error on request %d: %v", i, err)
			}
			if result != nil {
				t.Fatalf("expected no rate limit on request %d, got: %+v", i, result)
			}
		}

		// Next request should be rate limited
		result, err := rl.CheckAndIncrement(resourceKey, limits)
		if err != nil {
			t.Fatalf("unexpected error on excess request: %v", err)
		}
		if result == nil {
			t.Fatal("expected rate limit response, got nil")
		}

		// Wait for store TTL to expire (counters should expire automatically)
		time.Sleep(3 * time.Second)

		// Should now be able to make requests again (counters expired via store TTL)
		for i := 0; i < 2; i++ {
			result, err := rl.CheckAndIncrement(resourceKey, limits)
			if err != nil {
				t.Fatalf("unexpected error on request %d after TTL: %v", i, err)
			}
			if result != nil {
				t.Fatalf("expected no rate limit on request %d after TTL, got: %+v", i, result)
			}
		}
	})
}

func TestRateLimiterIntegrationInMemory(t *testing.T) {
	// Set InMemory TTL for integration testing
	oldTTL := os.Getenv("IMPOSTER_STORE_INMEMORY_TTL")
	os.Setenv("IMPOSTER_STORE_INMEMORY_TTL", "2") // 2 second TTL to match the rate limiter TTL test
	defer func() {
		if oldTTL == "" {
			os.Unsetenv("IMPOSTER_STORE_INMEMORY_TTL")
		} else {
			os.Setenv("IMPOSTER_STORE_INMEMORY_TTL", oldTTL)
		}
	}()

	storeProvider := setupTest(t)
	runRateLimiterIntegrationTest(t, storeProvider, "InMemory")
}

func TestRateLimiterIntegrationDynamoDB(t *testing.T) {
	storeProvider := setupDynamoDBIntegrationTest(t)
	runRateLimiterIntegrationTest(t, storeProvider, "DynamoDB")
}

func TestRateLimiterIntegrationRedis(t *testing.T) {
	storeProvider := setupRedisIntegrationTest(t)
	runRateLimiterIntegrationTest(t, storeProvider, "Redis")
}

// TestCrossStoreCompatibility tests that rate limiter behavior is consistent across store types
func TestCrossStoreCompatibility(t *testing.T) {
	// Test with multiple store backends if available
	storeTypes := []struct {
		name  string
		setup func(*testing.T) store.StoreProvider
	}{
		{"InMemory", setupTest},
		{"DynamoDB", setupDynamoDBIntegrationTest},
		{"Redis", setupRedisIntegrationTest},
	}

	for _, storeType := range storeTypes {
		t.Run(storeType.name, func(t *testing.T) {
			// Skip test if store is not available (will be handled by setup function)
			storeProvider := storeType.setup(t)

			rl := NewRateLimiter(storeProvider)
			defer rl.(*RateLimiterImpl).Stop()

			resourceKey := "GET:/compatibility-test"
			limits := []config.ConcurrencyLimit{
				{
					Limit: 3,
					Response: &config.Response{
						StatusCode: 429,
						Content:    "Rate limited",
					},
				},
			}

			// Test basic increment/decrement cycle

			// Should be able to increment up to limit
			result, err := rl.CheckAndIncrement(resourceKey, limits)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != nil {
				t.Fatalf("expected no rate limit, got: %+v", result)
			}

			// Should be able to decrement
			err = rl.Decrement(resourceKey)
			if err != nil {
				t.Fatalf("unexpected error on decrement: %v", err)
			}

			// Resource key generation should be consistent
			key1 := config.GenerateResourceKey("GET", "/test", nil)
			key2 := config.GenerateResourceKey("get", "/test", nil) // Different case
			key3 := config.GenerateResourceKey("", "/test", nil)    // Empty method

			if key1 != key2 {
				t.Errorf("resource key generation not case-insensitive: %s != %s", key1, key2)
			}
			if key3 != "*:/test" {
				t.Errorf("resource key generation for empty method incorrect: got %s, expected *:/test", key3)
			}
		})
	}
}

// BenchmarkRateLimiterStores benchmarks rate limiter performance with different store backends
func BenchmarkRateLimiterStores(b *testing.B) {
	storeTypes := []struct {
		name  string
		setup func() store.StoreProvider
	}{
		{
			"InMemory",
			func() store.StoreProvider {
				os.Setenv("IMPOSTER_STORE_DRIVER", "")
				store.InitStoreProvider()
				return store.GetStoreProvider()
			},
		},
	}

	// Add DynamoDB and Redis benchmarks if available
	if os.Getenv("IMPOSTER_DYNAMODB_TABLE") != "" {
		storeTypes = append(storeTypes, struct {
			name  string
			setup func() store.StoreProvider
		}{
			"DynamoDB",
			func() store.StoreProvider {
				os.Setenv("IMPOSTER_STORE_DRIVER", "store-dynamodb")
				store.InitStoreProvider()
				return store.GetStoreProvider()
			},
		})
	}

	if os.Getenv("REDIS_ADDR") != "" {
		storeTypes = append(storeTypes, struct {
			name  string
			setup func() store.StoreProvider
		}{
			"Redis",
			func() store.StoreProvider {
				os.Setenv("IMPOSTER_STORE_DRIVER", "store-redis")
				store.InitStoreProvider()
				return store.GetStoreProvider()
			},
		})
	}

	for _, storeType := range storeTypes {
		b.Run(storeType.name, func(b *testing.B) {
			storeProvider := storeType.setup()
			rl := NewRateLimiter(storeProvider)
			defer rl.(*RateLimiterImpl).Stop()

			resourceKey := "GET:/benchmark"
			limits := []config.ConcurrencyLimit{
				{
					Limit: 1000, // High limit to avoid rate limiting during benchmark
					Response: &config.Response{
						StatusCode: 429,
						Content:    "Rate limited",
					},
				},
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					rl.CheckAndIncrement(resourceKey, limits)
					rl.Decrement(resourceKey)
					i++
				}
			})
		})
	}
}
