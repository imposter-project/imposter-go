package store

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/imposter-project/imposter-go/pkg/logger"
)

type InMemoryStoreProvider struct {
	stores map[string]*storeData
	mu     sync.RWMutex
}

type storeData struct {
	data        map[string]interface{}
	expiryTimes map[string]time.Time
}

func (p *InMemoryStoreProvider) InitStores() {
	p.stores = make(map[string]*storeData)
}

func (p *InMemoryStoreProvider) GetValue(storeName, key string) (interface{}, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	store, ok := p.stores[storeName]
	if !ok {
		return nil, false
	}
	key = applyKeyPrefix(key)

	// Check if key has expired
	if expiry, hasExpiry := store.expiryTimes[key]; hasExpiry && time.Now().After(expiry) {
		delete(store.data, key)
		delete(store.expiryTimes, key)
		return nil, false
	}

	val, found := store.data[key]
	return val, found
}

func (p *InMemoryStoreProvider) StoreValue(storeName, key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.stores[storeName]; !ok {
		p.stores[storeName] = &storeData{
			data:        make(map[string]interface{}),
			expiryTimes: make(map[string]time.Time),
		}
	}
	key = applyKeyPrefix(key)
	p.stores[storeName].data[key] = value

	// Set TTL if configured
	ttl := getInMemoryTTL()
	if ttl > 0 {
		p.stores[storeName].expiryTimes[key] = time.Now().Add(ttl)
	}
}

func (p *InMemoryStoreProvider) GetAllValues(storeName, keyPrefix string) map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make(map[string]interface{})

	store, ok := p.stores[storeName]
	if !ok {
		return result
	}

	prefixToMatch := applyKeyPrefix(keyPrefix)
	expiredKeys := make([]string, 0)

	for k, v := range store.data {
		// Check if key has expired
		if expiry, hasExpiry := store.expiryTimes[k]; hasExpiry && time.Now().After(expiry) {
			expiredKeys = append(expiredKeys, k)
			continue
		}

		if strings.HasPrefix(k, prefixToMatch) {
			// Remove the global key prefix but keep the search prefix
			deprefixedKey := removeKeyPrefix(k)
			result[deprefixedKey] = v
		}
	}

	// Clean up expired keys
	for _, key := range expiredKeys {
		delete(store.data, key)
		delete(store.expiryTimes, key)
	}

	return result
}

func (p *InMemoryStoreProvider) DeleteValue(storeName, key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	store, ok := p.stores[storeName]
	if ok {
		key = applyKeyPrefix(key)
		delete(store.data, key)
		delete(store.expiryTimes, key)
	}
}

func (p *InMemoryStoreProvider) DeleteStore(storeName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.stores, storeName)
}

func (p *InMemoryStoreProvider) AtomicIncrement(storeName, key string, delta int64) (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.stores[storeName]; !ok {
		p.stores[storeName] = &storeData{
			data:        make(map[string]interface{}),
			expiryTimes: make(map[string]time.Time),
		}
	}

	key = applyKeyPrefix(key)

	// Check if key has expired
	store := p.stores[storeName]
	if expiry, hasExpiry := store.expiryTimes[key]; hasExpiry && time.Now().After(expiry) {
		delete(store.data, key)
		delete(store.expiryTimes, key)
	}

	// Get current value, default to 0 if not exists or not a number
	currentValue := int64(0)
	if val, exists := store.data[key]; exists {
		switch v := val.(type) {
		case int64:
			currentValue = v
		case int:
			currentValue = int64(v)
		case float64:
			currentValue = int64(v)
		default:
			// If existing value is not numeric, treat as 0
			currentValue = 0
		}
	}

	// Increment atomically
	newValue := currentValue + delta
	store.data[key] = newValue

	// Set expiration if this is a new key and InMemory TTL is configured
	if currentValue == 0 {
		ttl := getInMemoryTTL()
		if ttl > 0 {
			store.expiryTimes[key] = time.Now().Add(ttl)
		}
	}

	return newValue, nil
}

func (p *InMemoryStoreProvider) AtomicDecrement(storeName, key string, delta int64) (int64, error) {
	// Decrement is increment with negative delta
	return p.AtomicIncrement(storeName, key, -delta)
}

func getInMemoryTTL() time.Duration {
	ttlStr := os.Getenv("IMPOSTER_STORE_INMEMORY_TTL")
	if ttlStr == "" {
		return -1
	}
	ttl, err := strconv.ParseInt(ttlStr, 10, 64)
	if err != nil {
		logger.Errorf("invalid InMemory TTL value: %v", err)
		return -1
	}
	return time.Duration(ttl) * time.Second
}
