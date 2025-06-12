package ratelimiter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/internal/system"
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
	CheckAndIncrement(resourceKey string, limits []config.ConcurrencyLimit, instanceID string) (*config.ConcurrencyLimit, error)
	Decrement(resourceKey string, instanceID string) error
	Cleanup() error
}

// ResourceActivity represents per-server-instance resource activity data
type ResourceActivity struct {
	Count     int       `json:"count"`
	Timestamp time.Time `json:"timestamp"`
}

// RateLimiterImpl implements the RateLimiter interface
type RateLimiterImpl struct {
	storeProvider store.StoreProvider
	store         *store.Store
	instanceID    string
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
	instanceID := system.GenerateInstanceID()

	rl := &RateLimiterImpl{
		storeProvider: storeProvider,
		store:         store.Open(rateLimiterStoreName, nil),
		instanceID:    instanceID,
		ttl:           ttl,
		cleanupTicker: time.NewTicker(defaultCleanupTicker),
		cleanupDone:   make(chan bool),
	}

	// Start cleanup goroutine for inmemory store
	go rl.startCleanupRoutine()

	return rl
}

// CheckAndIncrement checks if the request should be rate limited and increments counter if not
func (rl *RateLimiterImpl) CheckAndIncrement(resourceKey string, limits []config.ConcurrencyLimit, instanceID string) (*config.ConcurrencyLimit, error) {
	if len(limits) == 0 {
		return nil, nil
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get current total active count
	totalActive, err := rl.getTotalActiveCount(resourceKey)
	if err != nil {
		logger.Warnf("failed to get total active count for resource %s: %v", resourceKey, err)
		// On error, allow the request to proceed
		return nil, nil
	}

	// Check if any limit is exceeded after incrementing
	futureTotal := totalActive + 1
	matchedLimit := rl.findMatchingLimit(futureTotal, limits)

	if matchedLimit != nil {
		logger.Infof("rate limit exceeded for resource %s: %d > %d", resourceKey, futureTotal, matchedLimit.Limit)
		return matchedLimit, nil
	}

	// Increment the counter
	err = rl.incrementActiveCount(resourceKey, instanceID)
	if err != nil {
		logger.Warnf("failed to increment active count for resource %s: %v", resourceKey, err)
		// On error, allow the request to proceed
		return nil, nil
	}

	return nil, nil
}

// Decrement decrements the active count for a resource and instance
func (rl *RateLimiterImpl) Decrement(resourceKey string, instanceID string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return rl.decrementActiveCount(resourceKey, instanceID)
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

// getTotalActiveCount gets the total active count across all instances for a resource
func (rl *RateLimiterImpl) getTotalActiveCount(resourceKey string) (int, error) {
	// Clean up expired activities first
	rl.cleanupExpiredActivities(resourceKey)

	// Get all resource activity data for this resource
	activityPrefix := rl.getActivityKeyPrefix(resourceKey)
	activities := rl.store.GetAllValues(activityPrefix)

	total := 0
	for _, value := range activities {
		if activityData, err := rl.parseResourceActivity(value); err == nil {
			// Check if activity data is still valid (not expired)
			if time.Since(activityData.Timestamp) <= rl.ttl {
				total += activityData.Count
			}
		}
	}

	return total, nil
}

// incrementActiveCount increments the active count for a specific server instance
func (rl *RateLimiterImpl) incrementActiveCount(resourceKey, instanceID string) error {
	activityKey := rl.getActivityKey(resourceKey, instanceID)

	// Get current activity data
	currentData := ResourceActivity{Count: 0, Timestamp: time.Now()}
	if value, exists := rl.store.GetValue(activityKey); exists {
		if existingData, err := rl.parseResourceActivity(value); err == nil {
			currentData.Count = existingData.Count
		}
	}

	// Increment and store
	currentData.Count++
	currentData.Timestamp = time.Now()

	dataBytes, err := json.Marshal(currentData)
	if err != nil {
		return fmt.Errorf("failed to marshal activity data: %w", err)
	}

	rl.store.StoreValue(activityKey, string(dataBytes))
	return nil
}

// decrementActiveCount decrements the active count for a specific server instance
func (rl *RateLimiterImpl) decrementActiveCount(resourceKey, instanceID string) error {
	activityKey := rl.getActivityKey(resourceKey, instanceID)

	// Get current activity data
	currentData := ResourceActivity{Count: 0, Timestamp: time.Now()}
	if value, exists := rl.store.GetValue(activityKey); exists {
		if existingData, err := rl.parseResourceActivity(value); err == nil {
			currentData = *existingData
		}
	}

	// Decrement (but don't go below 0)
	if currentData.Count > 0 {
		currentData.Count--
	}
	currentData.Timestamp = time.Now()

	if currentData.Count == 0 {
		// Remove the key if count reaches 0
		rl.store.DeleteValue(activityKey)
	} else {
		// Update the count
		dataBytes, err := json.Marshal(currentData)
		if err != nil {
			return fmt.Errorf("failed to marshal activity data: %w", err)
		}
		rl.store.StoreValue(activityKey, string(dataBytes))
	}

	return nil
}

// parseResourceActivity parses resource activity data from stored value
func (rl *RateLimiterImpl) parseResourceActivity(value interface{}) (*ResourceActivity, error) {
	var dataStr string
	switch v := value.(type) {
	case string:
		dataStr = v
	case []byte:
		dataStr = string(v)
	default:
		return nil, fmt.Errorf("invalid data type: %T", value)
	}

	var data ResourceActivity
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal activity data: %w", err)
	}

	return &data, nil
}

// cleanupExpiredActivities removes expired resource activity data
func (rl *RateLimiterImpl) cleanupExpiredActivities(resourceKey string) {
	activityPrefix := rl.getActivityKeyPrefix(resourceKey)
	activities := rl.store.GetAllValues(activityPrefix)

	for key, value := range activities {
		if activityData, err := rl.parseResourceActivity(value); err == nil {
			// Check if activity data is expired
			if time.Since(activityData.Timestamp) > rl.ttl {
				rl.store.DeleteValue(key)
				logger.Debugf("cleaned up expired resource activity: %s", key)
			}
		}
	}
}

// startCleanupRoutine starts the periodic cleanup routine
func (rl *RateLimiterImpl) startCleanupRoutine() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			if err := rl.Cleanup(); err != nil {
				logger.Warnf("cleanup failed: %v", err)
			}
		case <-rl.cleanupDone:
			return
		}
	}
}

// Cleanup performs cleanup of expired entries
func (rl *RateLimiterImpl) Cleanup() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get all keys in the rate limiter store
	allData := rl.store.GetAllValues("")

	for key, value := range allData {
		// Only process activity keys
		if !strings.Contains(key, activityKeyPrefix) {
			continue
		}

		if activityData, err := rl.parseResourceActivity(value); err == nil {
			// Check if activity data is expired
			if time.Since(activityData.Timestamp) > rl.ttl {
				rl.store.DeleteValue(key)
				logger.Debugf("cleaned up expired resource activity: %s", key)
			}
		}
	}

	return nil
}

// GenerateResourceKey generates a unique key for a resource
func GenerateResourceKey(method, name string) string {
	if method == "" {
		method = "*"
	}
	if name == "" {
		name = "*"
	}
	return fmt.Sprintf("%s:%s", strings.ToUpper(method), name)
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
