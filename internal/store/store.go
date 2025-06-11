package store

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/imposter-project/imposter-go/pkg/utils"

	"github.com/imposter-project/imposter-go/internal/config"
)

// StoreProvider interface defines the contract for store implementations
type StoreProvider interface {
	InitStores()
	GetValue(storeName, key string) (interface{}, bool)
	StoreValue(storeName, key string, value interface{})
	GetAllValues(storeName, keyPrefix string) map[string]interface{}
	DeleteValue(storeName, key string)
	DeleteStore(storeName string)
}

// Store represents a handle to a specific named store
type Store struct {
	name     string
	provider StoreProvider
}

// Open returns a handle to a specific store
func Open(storeName string, requestStore *Store) *Store {
	if storeName == "request" {
		return requestStore
	}
	return &Store{
		name:     storeName,
		provider: storeProvider,
	}
}

// GetValue retrieves a value from the store
func (s *Store) GetValue(key string) (interface{}, bool) {
	return s.provider.GetValue(s.name, key)
}

// StoreValue stores a value in the store
func (s *Store) StoreValue(key string, value interface{}) {
	s.provider.StoreValue(s.name, key, value)
}

// GetAllValues retrieves all values from the store with an optional prefix
func (s *Store) GetAllValues(keyPrefix string) map[string]interface{} {
	return s.provider.GetAllValues(s.name, keyPrefix)
}

// DeleteValue removes a value from the store
func (s *Store) DeleteValue(key string) {
	s.provider.DeleteValue(s.name, key)
}

// storeProvider is the global store provider
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

// GetStoreProvider returns the global store provider
func GetStoreProvider() StoreProvider {
	return storeProvider
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

	store := Open(storeName, nil)
	for k, v := range items {
		store.StoreValue(k, v)
	}
}

func preloadFromInline(storeName string, data map[string]interface{}) {
	logger.Infof("preloading store '%s' from inline data", storeName)
	store := Open(storeName, nil)
	for k, v := range data {
		store.StoreValue(k, v)
	}
}

// DeleteStore removes the entire store
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
