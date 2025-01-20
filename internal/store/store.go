package store

import (
	"encoding/json"
	"fmt"
	"github.com/imposter-project/imposter-go/pkg/utils"
	"os"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/logger"
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
					filePath, err := utils.ValidatePath(definition.PreloadFile, configDir)
					if err != nil {
						panic(fmt.Errorf("invalid preload file path: %s", definition.PreloadFile))
					}
					preloadFromFile(storeName, filePath)
				}
				if len(definition.PreloadData) > 0 {
					preloadFromInline(storeName, definition.PreloadData)
				}
			}
		}
	}
}

func preloadFromFile(storeName string, path string) {
	logger.Infof("preloading store '%s' from file: %s", storeName, path)
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Warnf("failed to read %s: %v", path, err)
		return
	}

	var items map[string]interface{}
	err = json.Unmarshal(data, &items)
	if err != nil {
		logger.Warnf("invalid JSON in %s: %v", path, err)
		return
	}

	for k, v := range items {
		storeProvider.StoreValue(storeName, k, v)
	}
}

func preloadFromInline(storeName string, data map[string]interface{}) {
	logger.Infof("preloading store '%s' from inline data", storeName)
	for k, v := range data {
		storeProvider.StoreValue(storeName, k, v)
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
