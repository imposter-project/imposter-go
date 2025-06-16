package store

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"golang.org/x/net/context"
)

type RedisStoreProvider struct {
	client *redis.Client
	ctx    context.Context
}

func (p *RedisStoreProvider) InitStores() {
	p.ctx = context.Background()
	p.client = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
}

func (p *RedisStoreProvider) GetValue(storeName, key string) (interface{}, bool) {
	key = applyKeyPrefix(key)
	redisKey := buildRedisKey(storeName, key)
	val, err := p.client.Get(p.ctx, redisKey).Result()
	if err == redis.Nil {
		return nil, false
	} else if err != nil {
		logger.Errorf("failed to get item: %v", err)
		return nil, false
	}
	var value interface{}
	if err := json.Unmarshal([]byte(val), &value); err != nil {
		logger.Errorf("failed to unmarshal value: %v", err)
		return nil, false
	}
	return value, true
}

func (p *RedisStoreProvider) StoreValue(storeName, key string, value interface{}) {
	key = applyKeyPrefix(key)
	redisKey := buildRedisKey(storeName, key)
	valueBytes, err := json.Marshal(value)
	if err != nil {
		logger.Errorf("failed to marshal value: %v", err)
		return
	}

	expiration := getExpiration()
	if expiration > 0 {
		err = p.client.Set(p.ctx, redisKey, valueBytes, expiration).Err()
	} else {
		err = p.client.Set(p.ctx, redisKey, valueBytes, 0).Err()
	}
	if err != nil {
		logger.Errorf("failed to set item: %v", err)
	}
}

func (p *RedisStoreProvider) GetAllValues(storeName, keyPrefix string) map[string]interface{} {
	keyPrefix = applyKeyPrefix(keyPrefix)
	searchPattern := buildRedisKey(storeName, keyPrefix) + "*"
	items := make(map[string]interface{})

	var cursor uint64
	for {
		keys, nextCursor, err := p.client.Scan(p.ctx, cursor, searchPattern, 100).Result()
		if err != nil {
			logger.Errorf("failed to scan keys: %v", err)
			return nil
		}

		for _, redisKey := range keys {
			val, err := p.client.Get(p.ctx, redisKey).Result()
			if err != nil {
				logger.Errorf("failed to get item %s: %v", redisKey, err)
				continue
			}
			var value interface{}
			if err := json.Unmarshal([]byte(val), &value); err != nil {
				logger.Errorf("failed to unmarshal value: %v", err)
				continue
			}
			// Extract the original key by removing storeName prefix
			key := extractKeyFromRedisKey(storeName, redisKey)
			deprefixedKey := removeKeyPrefix(key)
			items[deprefixedKey] = value
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return items
}

func (p *RedisStoreProvider) DeleteValue(storeName, key string) {
	key = applyKeyPrefix(key)
	redisKey := buildRedisKey(storeName, key)
	err := p.client.Del(p.ctx, redisKey).Err()
	if err != nil {
		logger.Errorf("failed to delete item: %v", err)
	}
}

func (p *RedisStoreProvider) DeleteStore(storeName string) {
	searchPattern := buildRedisKey(storeName, "*")

	// Use SCAN instead of KEYS for better performance
	var cursor uint64
	for {
		keys, nextCursor, err := p.client.Scan(p.ctx, cursor, searchPattern, 100).Result()
		if err != nil {
			logger.Errorf("failed to scan keys for store deletion: %v", err)
			return
		}

		if len(keys) > 0 {
			err = p.client.Del(p.ctx, keys...).Err()
			if err != nil {
				logger.Errorf("failed to delete keys: %v", err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

// buildRedisKey creates a Redis key from store name and key
func buildRedisKey(storeName, key string) string {
	return storeName + ":" + key
}

// extractKeyFromRedisKey extracts the original key from a Redis key
func extractKeyFromRedisKey(storeName, redisKey string) string {
	return strings.TrimPrefix(redisKey, storeName+":")
}

func getExpiration() time.Duration {
	expirationStr := os.Getenv("IMPOSTER_STORE_REDIS_EXPIRY")
	if expirationStr == "" {
		return -1
	}
	expiration, err := time.ParseDuration(expirationStr)
	if err != nil {
		logger.Errorf("invalid expiration duration: %v", err)
		return -1
	}
	return expiration
}

func (p *RedisStoreProvider) AtomicIncrement(storeName, key string, delta int64) (int64, error) {
	key = applyKeyPrefix(key)
	redisKey := buildRedisKey(storeName, key)

	// Use INCRBY for atomic increment
	newValue, err := p.client.IncrBy(p.ctx, redisKey, delta).Result()
	if err != nil {
		logger.Errorf("failed to atomic increment: %v", err)
		return 0, err
	}

	// Set expiration if configured and this is a new key (value equals delta)
	if newValue == delta {
		expiration := getExpiration()
		if expiration > 0 {
			err = p.client.Expire(p.ctx, redisKey, expiration).Err()
			if err != nil {
				logger.Warnf("failed to set expiration on atomic increment: %v", err)
			}
		}
	}

	return newValue, nil
}

func (p *RedisStoreProvider) AtomicDecrement(storeName, key string, delta int64) (int64, error) {
	// Use negative delta for decrement
	return p.AtomicIncrement(storeName, key, -delta)
}
