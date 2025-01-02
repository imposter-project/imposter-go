package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
)

// Store represents a key-value store with string keys and arbitrary values
type Store map[string]interface{}

type StoreProvider interface {
	InitStores()
	GetValue(storeName, key string) (interface{}, bool)
	StoreValue(storeName, key string, value interface{})
	GetAllValues(storeName, keyPrefix string) map[string]interface{}
	DeleteValue(storeName, key string)
	DeleteStore(storeName string)
}

var storeProvider StoreProvider

func InitStoreProvider() {
	driver := os.Getenv("IMPOSTER_STORE_DRIVER")
	switch driver {
	case "store-dynamodb":
		storeProvider = &DynamoDBStoreProvider{}
	case "store-redis":
		storeProvider = &RedisStoreProvider{}
	default:
		storeProvider = &InMemoryStoreProvider{}
	}
	storeProvider.InitStores()
}

func PreloadStores(configDir string, configs []config.Config) {
	for _, cfg := range configs {
		if cfg.System != nil && cfg.System.Stores != nil {
			for storeName, definition := range cfg.System.Stores {
				if definition.PreloadFile != "" {
					path := filepath.Join(configDir, definition.PreloadFile)
					fmt.Printf("Preloading store '%s' from file: %s\n", storeName, path)
					jsonBytes, err := os.ReadFile(path)
					if err != nil {
						fmt.Printf("Warning: failed to read %s: %v\n", path, err)
						continue
					}
					var jsonData map[string]interface{}
					if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
						fmt.Printf("Warning: invalid JSON in %s: %v\n", path, err)
						continue
					}
					for k, v := range jsonData {
						storeProvider.StoreValue(storeName, k, v)
					}
				}
				if len(definition.PreloadData) > 0 {
					fmt.Printf("Preloading store '%s' from inline data\n", storeName)
					for k, v := range definition.PreloadData {
						storeProvider.StoreValue(storeName, k, v)
					}
				}
			}
		}
	}
}

func GetValue(storeName, key string) (interface{}, bool) {
	return storeProvider.GetValue(storeName, key)
}

func StoreValue(storeName, key string, value interface{}) {
	storeProvider.StoreValue(storeName, key, value)
}

func GetAllValues(storeName, keyPrefix string) map[string]interface{} {
	return storeProvider.GetAllValues(storeName, keyPrefix)
}

func DeleteValue(storeName, key string) {
	storeProvider.DeleteValue(storeName, key)
}

func DeleteStore(storeName string) {
	storeProvider.DeleteStore(storeName)
}

func getStoreKeyPrefix() string {
	return os.Getenv("IMPOSTER_STORE_KEY_PREFIX")
}

func applyKeyPrefix(key string) string {
	prefix := getStoreKeyPrefix()
	if prefix != "" {
		return prefix + "." + key
	}
	return key
}

func removeKeyPrefix(key string) string {
	prefix := getStoreKeyPrefix()
	if prefix != "" {
		return strings.TrimPrefix(key, prefix+".")
	}
	return key
}
