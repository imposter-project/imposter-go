package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gatehill/imposter-go/internal/config"
)

type storeData struct {
	data map[string]interface{}
}

var stores = make(map[string]*storeData)

func InitStores() {
	// No-op for now
}

func PreloadStores(configDir string, configs []config.Config) {
	for _, cfg := range configs {
		if cfg.System != nil && cfg.System.Stores != nil {
			for storeName, definition := range cfg.System.Stores {
				if _, ok := stores[storeName]; !ok {
					stores[storeName] = &storeData{data: make(map[string]interface{})}
				}
				if definition.PreloadFile != "" {
					path := filepath.Join(configDir, definition.PreloadFile)
					fmt.Printf("Preloading store '%s' from file: %s\n", storeName, path)
					jsonBytes, err := ioutil.ReadFile(path)
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
						stores[storeName].data[k] = v
					}
				}
				if len(definition.PreloadData) > 0 {
					fmt.Printf("Preloading store '%s' from inline data\n", storeName)
					for k, v := range definition.PreloadData {
						stores[storeName].data[k] = v
					}
				}
			}
		}
	}
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

func GetValue(storeName, key string) (interface{}, bool) {
	store, ok := stores[storeName]
	if !ok {
		return nil, false
	}
	key = applyKeyPrefix(key)
	val, found := store.data[key]
	return val, found
}

func StoreValue(storeName, key string, value interface{}) {
	if _, ok := stores[storeName]; !ok {
		stores[storeName] = &storeData{data: make(map[string]interface{})}
	}
	key = applyKeyPrefix(key)
	stores[storeName].data[key] = value
}

func GetAllValues(storeName, searchPrefix string) map[string]interface{} {
	store, ok := stores[storeName]
	if !ok {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range store.data {
		if strings.HasPrefix(k, searchPrefix) {
			deprefixedKey := removeKeyPrefix(k)
			result[deprefixedKey] = v
		}
	}
	return result
}

func DeleteValue(storeName, key string) {
	store, ok := stores[storeName]
	if ok {
		key = applyKeyPrefix(key)
		delete(store.data, key)
	}
}

func DeleteStore(storeName string) {
	delete(stores, storeName)
}
