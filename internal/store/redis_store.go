package store

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gatehill/imposter-go/internal/config"
	"github.com/go-redis/redis/v8"
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

func (p *RedisStoreProvider) PreloadStores(configDir string, configs []config.Config) {
	// No-op for now
}

func (p *RedisStoreProvider) GetValue(storeName, key string) (interface{}, bool) {
	key = applyKeyPrefix(key)
	val, err := p.client.HGet(p.ctx, storeName, key).Result()
	if err == redis.Nil {
		return nil, false
	} else if err != nil {
		fmt.Printf("Failed to get item: %v\n", err)
		return nil, false
	}
	var value interface{}
	if err := json.Unmarshal([]byte(val), &value); err != nil {
		fmt.Printf("Failed to unmarshal value: %v\n", err)
		return nil, false
	}
	return value, true
}

func (p *RedisStoreProvider) StoreValue(storeName, key string, value interface{}) {
	key = applyKeyPrefix(key)
	valueBytes, err := json.Marshal(value)
	if err != nil {
		fmt.Printf("Failed to marshal value: %v\n", err)
		return
	}
	expiration := getExpiration()
	err = p.client.HSet(p.ctx, storeName, key, valueBytes).Err()
	if err != nil {
		fmt.Printf("Failed to set item: %v\n", err)
		return
	}
	err = p.client.Expire(p.ctx, storeName+":"+key, expiration).Err()
	if err != nil {
		fmt.Printf("Failed to set expiration: %v\n", err)
	}
}

func (p *RedisStoreProvider) GetAllValues(storeName, keyPrefix string) map[string]interface{} {
	keyPrefix = applyKeyPrefix(keyPrefix)
	items := make(map[string]interface{})
	vals, err := p.client.HGetAll(p.ctx, storeName).Result()
	if err != nil {
		fmt.Printf("Failed to get items: %v\n", err)
		return nil
	}
	for key, val := range vals {
		if strings.HasPrefix(key, keyPrefix) {
			var value interface{}
			if err := json.Unmarshal([]byte(val), &value); err != nil {
				fmt.Printf("Failed to unmarshal value: %v\n", err)
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
		fmt.Printf("Failed to delete item: %v\n", err)
	}
}

func (p *RedisStoreProvider) DeleteStore(storeName string) {
	err := p.client.Del(p.ctx, storeName).Err()
	if err != nil {
		fmt.Printf("Failed to delete store: %v\n", err)
	}
}

func getExpiration() time.Duration {
	expirationStr := os.Getenv("IMPOSTER_STORE_REDIS_EXPIRY")
	if expirationStr == "" {
		return 30 * time.Minute
	}
	expiration, err := time.ParseDuration(expirationStr)
	if err != nil {
		fmt.Printf("Invalid expiration duration: %v\n", err)
		return 30 * time.Minute
	}
	return expiration
}
