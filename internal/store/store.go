package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

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

func GetValue(storeName, key string) (interface{}, bool) {
	store, ok := stores[storeName]
	if !ok {
		return nil, false
	}
	val, found := store.data[key]
	return val, found
}

func StoreValue(storeName, key, value string) {
	if _, ok := stores[storeName]; !ok {
		stores[storeName] = &storeData{data: make(map[string]interface{})}
	}
	stores[storeName].data[key] = value
}
