package store

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/imposter-project/imposter-go/internal/logger"
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
	val, err := p.client.HGet(p.ctx, storeName, key).Result()
	if err == redis.Nil {
		return nil, false
	} else if err != nil {
		if err != redis.Nil {
			logger.Errorf("failed to get item: %v", err)
		}
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
	valueBytes, err := json.Marshal(value)
	if err != nil {
		logger.Errorf("failed to marshal value: %v", err)
		return
	}
	expiration := getExpiration()
	err = p.client.HSet(p.ctx, storeName, key, valueBytes).Err()
	if err != nil {
		logger.Errorf("failed to set item: %v", err)
		return
	}
	err = p.client.Expire(p.ctx, storeName+":"+key, expiration).Err()
	if err != nil {
		logger.Errorf("failed to set expiration: %v", err)
	}
}

func (p *RedisStoreProvider) GetAllValues(storeName, keyPrefix string) map[string]interface{} {
	keyPrefix = applyKeyPrefix(keyPrefix)
	items := make(map[string]interface{})
	vals, err := p.client.HGetAll(p.ctx, storeName).Result()
	if err != nil {
		logger.Errorf("failed to get items: %v", err)
		return nil
	}
	for key, val := range vals {
		if strings.HasPrefix(key, keyPrefix) {
			var value interface{}
			if err := json.Unmarshal([]byte(val), &value); err != nil {
				logger.Errorf("failed to unmarshal value: %v", err)
				continue
			}
			deprefixedKey := removeKeyPrefix(key)
			items[deprefixedKey] = value
		}
	}
	return items
}

func (p *RedisStoreProvider) DeleteValue(storeName, key string) {
	key = applyKeyPrefix(key)
	err := p.client.HDel(p.ctx, storeName, key).Err()
	if err != nil {
		logger.Errorf("failed to delete item: %v", err)
	}
}

func (p *RedisStoreProvider) DeleteStore(storeName string) {
	err := p.client.Del(p.ctx, storeName).Err()
	if err != nil {
		logger.Errorf("failed to delete store: %v", err)
	}
}

func getExpiration() time.Duration {
	expirationStr := os.Getenv("IMPOSTER_STORE_REDIS_EXPIRY")
	if expirationStr == "" {
		return 30 * time.Minute
	}
	expiration, err := time.ParseDuration(expirationStr)
	if err != nil {
		logger.Errorf("invalid expiration duration: %v", err)
		return 30 * time.Minute
	}
	return expiration
}
