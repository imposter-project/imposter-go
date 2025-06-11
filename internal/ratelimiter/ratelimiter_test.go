package ratelimiter

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
)

func setupTest(t *testing.T) store.StoreProvider {
	// Set up a clean environment variable to ensure inmemory store
	oldDriver := os.Getenv("IMPOSTER_STORE_DRIVER")
	os.Setenv("IMPOSTER_STORE_DRIVER", "")

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

func TestRateLimiter_CheckAndIncrement_NoLimits(t *testing.T) {
	storeProvider := setupTest(t)

	rl := NewRateLimiter(storeProvider)
	defer rl.(*RateLimiterImpl).Stop()

	resourceKey := "GET:/test"
	instanceID := "test-instance"
	limits := []config.ConcurrencyLimit{}

	result, err := rl.CheckAndIncrement(resourceKey, limits, instanceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected no rate limit, got: %+v", result)
	}
}

func TestRateLimiter_CheckAndIncrement_BelowLimit(t *testing.T) {
	storeProvider := setupTest(t)

	rl := NewRateLimiter(storeProvider)
	defer rl.(*RateLimiterImpl).Stop()

	resourceKey := "GET:/test"
	instanceID := "test-instance"
	limits := []config.ConcurrencyLimit{
		{
			Limit: 5,
			Response: &config.Response{
				StatusCode: 429,
				Content:    "Rate limited",
			},
		},
	}

	// First 4 requests should pass
	for i := 0; i < 4; i++ {
		result, err := rl.CheckAndIncrement(resourceKey, limits, fmt.Sprintf("%s-%d", instanceID, i))
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i, err)
		}
		if result != nil {
			t.Fatalf("expected no rate limit on request %d, got: %+v", i, result)
		}
	}
}

func TestRateLimiter_CheckAndIncrement_ExceedsLimit(t *testing.T) {
	storeProvider := setupTest(t)

	rl := NewRateLimiter(storeProvider)
	defer rl.(*RateLimiterImpl).Stop()

	resourceKey := "GET:/test"
	instanceID := "test-instance"
	limits := []config.ConcurrencyLimit{
		{
			Limit: 3,
			Response: &config.Response{
				StatusCode: 429,
				Content:    "Rate limited",
			},
		},
	}

	// First 3 requests should pass (up to the limit)
	for i := 0; i < 3; i++ {
		result, err := rl.CheckAndIncrement(resourceKey, limits, fmt.Sprintf("%s-%d", instanceID, i))
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i, err)
		}
		if result != nil {
			t.Fatalf("expected no rate limit on request %d, got: %+v", i, result)
		}
	}

	// Fourth request should be rate limited
	result, err := rl.CheckAndIncrement(resourceKey, limits, fmt.Sprintf("%s-4", instanceID))
	if err != nil {
		t.Fatalf("unexpected error on rate limited request: %v", err)
	}
	if result == nil {
		t.Fatal("expected rate limit response, got nil")
	}
	if result.Response.StatusCode != 429 {
		t.Fatalf("expected status code 429, got %d", result.Response.StatusCode)
	}
}

func TestRateLimiter_CheckAndIncrement_MultipleLimits(t *testing.T) {
	storeProvider := setupTest(t)

	rl := NewRateLimiter(storeProvider)
	defer rl.(*RateLimiterImpl).Stop()

	resourceKey := "GET:/test"
	instanceID := "test-instance"
	limits := []config.ConcurrencyLimit{
		{
			Limit: 3,
			Response: &config.Response{
				StatusCode: 503,
				Content:    "Server overloaded",
			},
		},
		{
			Limit: 5,
			Response: &config.Response{
				StatusCode: 429,
				Content:    "Too many requests",
			},
		},
	}

	// First 3 requests should pass (up to first limit)
	for i := 0; i < 3; i++ {
		result, err := rl.CheckAndIncrement(resourceKey, limits, fmt.Sprintf("%s-%d", instanceID, i))
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i, err)
		}
		if result != nil {
			t.Fatalf("expected no rate limit on request %d, got: %+v", i, result)
		}
	}

	// Fourth request should hit the first limit (exceeds limit: 3)
	result, err := rl.CheckAndIncrement(resourceKey, limits, fmt.Sprintf("%s-4", instanceID))
	if err != nil {
		t.Fatalf("unexpected error on rate limited request: %v", err)
	}
	if result == nil {
		t.Fatal("expected rate limit response, got nil")
	}
	if result.Response.StatusCode != 503 {
		t.Fatalf("expected status code 503, got %d", result.Response.StatusCode)
	}

	// All subsequent requests should continue to hit the same limit since count doesn't increase
	// when rate limited
	for i := 5; i < 8; i++ {
		result, err := rl.CheckAndIncrement(resourceKey, limits, fmt.Sprintf("%s-%d", instanceID, i))
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i, err)
		}
		if result == nil {
			t.Fatalf("expected rate limit response on request %d, got nil", i)
		}
		if result.Response.StatusCode != 503 {
			t.Fatalf("expected status code 503 on request %d, got %d", i, result.Response.StatusCode)
		}
	}
}

func TestRateLimiter_Decrement(t *testing.T) {
	storeProvider := setupTest(t)

	rl := NewRateLimiter(storeProvider)
	defer rl.(*RateLimiterImpl).Stop()

	resourceKey := "GET:/test"
	instanceID := "test-instance"
	limits := []config.ConcurrencyLimit{
		{
			Limit: 2,
			Response: &config.Response{
				StatusCode: 429,
				Content:    "Rate limited",
			},
		},
	}

	// Increment to the limit
	for i := 0; i < 2; i++ {
		result, err := rl.CheckAndIncrement(resourceKey, limits, fmt.Sprintf("%s-%d", instanceID, i))
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i, err)
		}
		if result != nil {
			t.Fatalf("expected no rate limit on request %d, got: %+v", i, result)
		}
	}

	// Decrement one request
	err := rl.Decrement(resourceKey, fmt.Sprintf("%s-0", instanceID))
	if err != nil {
		t.Fatalf("unexpected error on decrement: %v", err)
	}

	// Should be able to add another request now
	result, err := rl.CheckAndIncrement(resourceKey, limits, fmt.Sprintf("%s-new", instanceID))
	if err != nil {
		t.Fatalf("unexpected error on new request: %v", err)
	}
	if result != nil {
		t.Fatalf("expected no rate limit after decrement, got: %+v", result)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	storeProvider := setupTest(t)

	rl := NewRateLimiter(storeProvider)
	defer rl.(*RateLimiterImpl).Stop()

	resourceKey := "GET:/test"
	limits := []config.ConcurrencyLimit{
		{
			Limit: 10,
			Response: &config.Response{
				StatusCode: 429,
				Content:    "Rate limited",
			},
		},
	}

	numGoroutines := 20
	numRequestsPerGoroutine := 5
	var wg sync.WaitGroup
	var rateLimitedCount int32
	var mu sync.Mutex

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numRequestsPerGoroutine; j++ {
				instanceID := fmt.Sprintf("goroutine-%d-request-%d", goroutineID, j)
				result, err := rl.CheckAndIncrement(resourceKey, limits, instanceID)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}

				if result != nil {
					mu.Lock()
					rateLimitedCount++
					mu.Unlock()
				} else {
					// Simulate some work, then decrement
					time.Sleep(10 * time.Millisecond)
					if err := rl.Decrement(resourceKey, instanceID); err != nil {
						t.Errorf("unexpected error on decrement: %v", err)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// We should have some rate limited requests since we're trying to exceed the limit
	mu.Lock()
	defer mu.Unlock()
	if rateLimitedCount == 0 {
		t.Error("expected some requests to be rate limited, but none were")
	}

	t.Logf("Rate limited %d out of %d total requests", rateLimitedCount, numGoroutines*numRequestsPerGoroutine)
}

func TestRateLimiter_TTLCleanup(t *testing.T) {
	storeProvider := setupTest(t)

	// Create rate limiter with short TTL for testing
	rl := NewRateLimiterWithTTL(storeProvider, 100*time.Millisecond)
	defer rl.(*RateLimiterImpl).Stop()

	resourceKey := "GET:/test"
	instanceID := "test-instance"
	limits := []config.ConcurrencyLimit{
		{
			Limit: 5,
			Response: &config.Response{
				StatusCode: 429,
				Content:    "Rate limited",
			},
		},
	}

	// Add some requests
	for i := 0; i < 3; i++ {
		result, err := rl.CheckAndIncrement(resourceKey, limits, fmt.Sprintf("%s-%d", instanceID, i))
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i, err)
		}
		if result != nil {
			t.Fatalf("expected no rate limit on request %d, got: %+v", i, result)
		}
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Run cleanup
	if err := rl.Cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// All requests should now pass since expired entries were cleaned up
	for i := 0; i < 4; i++ {
		result, err := rl.CheckAndIncrement(resourceKey, limits, fmt.Sprintf("%s-new-%d", instanceID, i))
		if err != nil {
			t.Fatalf("unexpected error on request %d after cleanup: %v", i, err)
		}
		if result != nil {
			t.Fatalf("expected no rate limit on request %d after cleanup, got: %+v", i, result)
		}
	}
}

func TestGenerateResourceKey(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		expected string
	}{
		{"GET", "/test", "GET:/test"},
		{"POST", "/api/users", "POST:/api/users"},
		{"", "/test", "*:/test"},
		{"get", "/test", "GET:/test"},
	}

	for _, test := range tests {
		result := GenerateResourceKey(test.method, test.path)
		if result != test.expected {
			t.Errorf("GenerateResourceKey(%q, %q) = %q, expected %q",
				test.method, test.path, result, test.expected)
		}
	}
}

func TestFindMatchingLimit(t *testing.T) {
	rl := &RateLimiterImpl{}

	limits := []config.ConcurrencyLimit{
		{
			Limit:    10,
			Response: &config.Response{StatusCode: 429},
		},
		{
			Limit:    5,
			Response: &config.Response{StatusCode: 503},
		},
		{
			Limit:    15,
			Response: &config.Response{StatusCode: 502},
		},
	}

	tests := []struct {
		count          int
		expectedStatus int
		expectNil      bool
	}{
		{3, 0, true},     // Below all limits
		{5, 0, true},     // At first limit (5), should be allowed
		{6, 503, false},  // Exceeds first limit (5)
		{8, 503, false},  // Still exceeds first limit (5)
		{10, 503, false}, // At second limit (10), still matches first
		{11, 429, false}, // Exceeds second limit (10)
		{12, 429, false}, // Still exceeds second limit (10)
		{15, 429, false}, // At third limit (15), still matches second
		{16, 502, false}, // Exceeds third limit (15)
		{20, 502, false}, // Still exceeds third limit (15)
	}

	for _, test := range tests {
		result := rl.findMatchingLimit(test.count, limits)
		if test.expectNil {
			if result != nil {
				t.Errorf("findMatchingLimit(%d) expected nil, got %+v", test.count, result)
			}
		} else {
			if result == nil {
				t.Errorf("findMatchingLimit(%d) expected non-nil, got nil", test.count)
			} else if result.Response.StatusCode != test.expectedStatus {
				t.Errorf("findMatchingLimit(%d) expected status %d, got %d",
					test.count, test.expectedStatus, result.Response.StatusCode)
			}
		}
	}
}
