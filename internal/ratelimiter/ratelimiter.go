package ratelimiter

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

const (
	defaultTTL           = 5 * time.Minute
	defaultCleanupTicker = 1 * time.Minute
	rateLimiterStoreName = "rate_limiter"
	activityKeyPrefix    = "activity:"
)

var (
	globalRateLimiter RateLimiter
	globalRLMutex     sync.Mutex
)

// RateLimiter interface defines the contract for rate limiting functionality
type RateLimiter interface {
	CheckAndIncrement(resourceKey string, limits []config.ConcurrencyLimit) (*config.ConcurrencyLimit, error)
	Decrement(resourceKey string) error
}

// RateLimiterImpl implements the RateLimiter interface
type RateLimiterImpl struct {
	storeProvider store.StoreProvider
	store         *store.Store
	ttl           time.Duration
	cleanupTicker *time.Ticker
	cleanupDone   chan bool
	mu            sync.RWMutex
}

// GetGlobalRateLimiter returns the global rate limiter instance, initializing it if needed
func GetGlobalRateLimiter() RateLimiter {
	// First check without locking (fast path)
	if globalRateLimiter != nil {
		return globalRateLimiter
	}

	globalRLMutex.Lock()
	defer globalRLMutex.Unlock()

	// Check again after acquiring the lock
	if globalRateLimiter == nil {
		storeProvider := store.GetStoreProvider()
		globalRateLimiter = NewRateLimiter(storeProvider)
	}
	return globalRateLimiter
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(storeProvider store.StoreProvider) RateLimiter {
	return NewRateLimiterWithTTL(storeProvider, getTTLFromEnv())
}

// NewRateLimiterWithTTL creates a new rate limiter instance with custom TTL
func NewRateLimiterWithTTL(storeProvider store.StoreProvider, ttl time.Duration) RateLimiter {
	rl := &RateLimiterImpl{
		storeProvider: storeProvider,
		store:         store.Open(rateLimiterStoreName, nil),
		ttl:           ttl,
		cleanupTicker: time.NewTicker(defaultCleanupTicker),
		cleanupDone:   make(chan bool),
	}
	return rl
}

// CheckAndIncrement checks if the request should be rate limited and increments counter if not
func (rl *RateLimiterImpl) CheckAndIncrement(resourceKey string, limits []config.ConcurrencyLimit) (*config.ConcurrencyLimit, error) {
	if len(limits) == 0 {
		return nil, nil
	}

	// Use single atomic counter per resource - no instance ID needed
	counterKey := rl.getResourceCounterKey(resourceKey)
	newCount, err := rl.store.AtomicIncrement(counterKey, 1)
	if err != nil {
		logger.Warnf("failed to atomic increment for resource %s: %v", resourceKey, err)
		// On error, allow the request to proceed
		return nil, nil
	}

	// Check if any limit is exceeded after incrementing
	matchedLimit := rl.findMatchingLimit(int(newCount), limits)

	if matchedLimit != nil {
		// Rate limit exceeded - rollback the increment
		_, rollbackErr := rl.store.AtomicDecrement(counterKey, 1)
		if rollbackErr != nil {
			logger.Warnf("failed to rollback increment for resource %s: %v", resourceKey, rollbackErr)
		}
		logger.Infof("rate limit exceeded for resource %s: %d > %d", resourceKey, newCount, matchedLimit.Limit)
		return matchedLimit, nil
	}

	return nil, nil
}

// Decrement decrements the active count for a resource
func (rl *RateLimiterImpl) Decrement(resourceKey string) error {
	counterKey := rl.getResourceCounterKey(resourceKey)
	_, err := rl.store.AtomicDecrement(counterKey, 1)
	if err != nil {
		logger.Warnf("failed to atomic decrement for resource %s: %v", resourceKey, err)
		return err
	}
	return nil
}

// findMatchingLimit finds the highest matching limit using "greater than or equal to" logic
func (rl *RateLimiterImpl) findMatchingLimit(currentCount int, limits []config.ConcurrencyLimit) *config.ConcurrencyLimit {
	// Sort limits by threshold (ascending) for proper matching
	sortedLimits := make([]config.ConcurrencyLimit, len(limits))
	copy(sortedLimits, limits)
	sort.Slice(sortedLimits, func(i, j int) bool {
		return sortedLimits[i].Limit < sortedLimits[j].Limit
	})

	// Find the highest matching limit (> logic)
	var matchedLimit *config.ConcurrencyLimit
	for _, limit := range sortedLimits {
		if currentCount > limit.Limit {
			matchedLimit = &limit
		}
	}

	return matchedLimit
}

// getTTLFromEnv gets TTL from environment variable or returns default
func getTTLFromEnv() time.Duration {
	ttlStr := os.Getenv("IMPOSTER_RATE_LIMITER_TTL")
	if ttlStr == "" {
		return defaultTTL
	}

	if ttlSeconds, err := strconv.Atoi(ttlStr); err == nil {
		return time.Duration(ttlSeconds) * time.Second
	}

	return defaultTTL
}

// Key generation helpers
func (rl *RateLimiterImpl) getResourceCounterKey(resourceKey string) string {
	return fmt.Sprintf("counter:%s", resourceKey)
}

func (rl *RateLimiterImpl) getActivityKeyPrefix(resourceKey string) string {
	return fmt.Sprintf("%s%s:", activityKeyPrefix, resourceKey)
}

func (rl *RateLimiterImpl) getActivityKey(resourceKey, instanceID string) string {
	return fmt.Sprintf("%s%s:%s", activityKeyPrefix, resourceKey, instanceID)
}

// Stop stops the cleanup routine and releases resources
func (rl *RateLimiterImpl) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
	if rl.cleanupDone != nil {
		close(rl.cleanupDone)
	}
}
