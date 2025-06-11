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
	"github.com/imposter-project/imposter-go/pkg/logger"
)

const (
	defaultTTL           = 5 * time.Minute
	defaultCleanupTicker = 1 * time.Minute
	rateLimiterStoreName = "rate_limiter"
	instanceKeyPrefix    = "instance:"
	totalKeyPrefix       = "total:"
	instancesKeyPrefix   = "instances:"
	timestampKeySuffix   = ":timestamp"
)

// RateLimiter interface defines the contract for rate limiting functionality
type RateLimiter interface {
	CheckAndIncrement(resourceKey string, limits []config.ConcurrencyLimit, instanceID string) (*config.ConcurrencyLimit, error)
	Decrement(resourceKey string, instanceID string) error
	Cleanup() error
}

// InstanceData represents per-instance concurrency data
type InstanceData struct {
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

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(storeProvider store.StoreProvider) RateLimiter {
	return NewRateLimiterWithTTL(storeProvider, getTTLFromEnv())
}

// NewRateLimiterWithTTL creates a new rate limiter instance with custom TTL
func NewRateLimiterWithTTL(storeProvider store.StoreProvider, ttl time.Duration) RateLimiter {
	instanceID := generateInstanceID()

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
	// Clean up expired instances first
	rl.cleanupExpiredInstances(resourceKey)

	// Get all instance data for this resource
	instancePrefix := rl.getInstanceKeyPrefix(resourceKey)
	instances := rl.store.GetAllValues(instancePrefix)

	total := 0
	for key, value := range instances {
		// Skip timestamp keys
		if strings.HasSuffix(key, timestampKeySuffix) {
			continue
		}

		if instanceData, err := rl.parseInstanceData(value); err == nil {
			// Check if instance data is still valid (not expired)
			if time.Since(instanceData.Timestamp) <= rl.ttl {
				total += instanceData.Count
			}
		}
	}

	return total, nil
}

// incrementActiveCount increments the active count for a specific instance
func (rl *RateLimiterImpl) incrementActiveCount(resourceKey, instanceID string) error {
	instanceKey := rl.getInstanceKey(resourceKey, instanceID)

	// Get current instance data
	currentData := InstanceData{Count: 0, Timestamp: time.Now()}
	if value, exists := rl.store.GetValue(instanceKey); exists {
		if existingData, err := rl.parseInstanceData(value); err == nil {
			currentData.Count = existingData.Count
		}
	}

	// Increment and store
	currentData.Count++
	currentData.Timestamp = time.Now()

	dataBytes, err := json.Marshal(currentData)
	if err != nil {
		return fmt.Errorf("failed to marshal instance data: %w", err)
	}

	rl.store.StoreValue(instanceKey, string(dataBytes))
	return nil
}

// decrementActiveCount decrements the active count for a specific instance
func (rl *RateLimiterImpl) decrementActiveCount(resourceKey, instanceID string) error {
	instanceKey := rl.getInstanceKey(resourceKey, instanceID)

	// Get current instance data
	currentData := InstanceData{Count: 0, Timestamp: time.Now()}
	if value, exists := rl.store.GetValue(instanceKey); exists {
		if existingData, err := rl.parseInstanceData(value); err == nil {
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
		rl.store.DeleteValue(instanceKey)
	} else {
		// Update the count
		dataBytes, err := json.Marshal(currentData)
		if err != nil {
			return fmt.Errorf("failed to marshal instance data: %w", err)
		}
		rl.store.StoreValue(instanceKey, string(dataBytes))
	}

	return nil
}

// parseInstanceData parses instance data from stored value
func (rl *RateLimiterImpl) parseInstanceData(value interface{}) (*InstanceData, error) {
	var dataStr string
	switch v := value.(type) {
	case string:
		dataStr = v
	case []byte:
		dataStr = string(v)
	default:
		return nil, fmt.Errorf("invalid data type: %T", value)
	}

	var data InstanceData
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance data: %w", err)
	}

	return &data, nil
}

// cleanupExpiredInstances removes expired instance data
func (rl *RateLimiterImpl) cleanupExpiredInstances(resourceKey string) {
	instancePrefix := rl.getInstanceKeyPrefix(resourceKey)
	instances := rl.store.GetAllValues(instancePrefix)

	for key, value := range instances {
		// Skip timestamp keys
		if strings.HasSuffix(key, timestampKeySuffix) {
			continue
		}

		if instanceData, err := rl.parseInstanceData(value); err == nil {
			// Check if instance data is expired
			if time.Since(instanceData.Timestamp) > rl.ttl {
				rl.store.DeleteValue(key)
				logger.Debugf("cleaned up expired instance data: %s", key)
			}
		}
	}
}

// startCleanupRoutine starts the periodic cleanup routine
func (rl *RateLimiterImpl) startCleanupRoutine() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.Cleanup()
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
		// Only process instance keys
		if !strings.Contains(key, instanceKeyPrefix) {
			continue
		}

		if instanceData, err := rl.parseInstanceData(value); err == nil {
			// Check if instance data is expired
			if time.Since(instanceData.Timestamp) > rl.ttl {
				rl.store.DeleteValue(key)
				logger.Debugf("cleaned up expired instance data: %s", key)
			}
		}
	}

	return nil
}

// generateInstanceID generates a unique instance ID for this server instance
func generateInstanceID() string {
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s-%d-%d", hostname, pid, timestamp)
}

// generateResourceKey generates a unique key for a resource
func GenerateResourceKey(method, path string) string {
	if method == "" {
		method = "*"
	}
	return fmt.Sprintf("%s:%s", strings.ToUpper(method), path)
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
func (rl *RateLimiterImpl) getInstanceKeyPrefix(resourceKey string) string {
	return fmt.Sprintf("%s%s:", instanceKeyPrefix, resourceKey)
}

func (rl *RateLimiterImpl) getInstanceKey(resourceKey, instanceID string) string {
	return fmt.Sprintf("%s%s:%s", instanceKeyPrefix, resourceKey, instanceID)
}

func (rl *RateLimiterImpl) getTotalKey(resourceKey string) string {
	return fmt.Sprintf("%s%s", totalKeyPrefix, resourceKey)
}

func (rl *RateLimiterImpl) getInstancesKey(resourceKey string) string {
	return fmt.Sprintf("%s%s", instancesKeyPrefix, resourceKey)
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
