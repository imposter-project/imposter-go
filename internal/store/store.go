package store

import (
	"os"
	"strings"

	"github.com/gatehill/imposter-go/internal/config"
)

type StoreProvider interface {
	InitStores()
	PreloadStores(configDir string, configs []config.Config)
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
	default:
		storeProvider = &InMemoryStoreProvider{}
	}
	storeProvider.InitStores()
}

func PreloadStores(configDir string, configs []config.Config) {
	storeProvider.PreloadStores(configDir, configs)
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
