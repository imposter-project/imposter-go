package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/gatehill/imposter-go/internal/config"
)

type InMemoryStoreProvider struct {
	stores map[string]*storeData
}

type storeData struct {
	data map[string]interface{}
}

func (p *InMemoryStoreProvider) InitStores() {
	p.stores = make(map[string]*storeData)
}

func (p *InMemoryStoreProvider) PreloadStores(configDir string, configs []config.Config) {
	for _, cfg := range configs {
		if cfg.System != nil && cfg.System.Stores != nil {
			for storeName, definition := range cfg.System.Stores {
				if _, ok := p.stores[storeName]; !ok {
					p.stores[storeName] = &storeData{data: make(map[string]interface{})}
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
						p.stores[storeName].data[k] = v
					}
				}
				if len(definition.PreloadData) > 0 {
					fmt.Printf("Preloading store '%s' from inline data\n", storeName)
					for k, v := range definition.PreloadData {
						p.stores[storeName].data[k] = v
					}
				}
			}
		}
	}
}

func (p *InMemoryStoreProvider) GetValue(storeName, key string) (interface{}, bool) {
	store, ok := p.stores[storeName]
	if !ok {
		return nil, false
	}
	key = applyKeyPrefix(key)
	val, found := store.data[key]
	return val, found
}

func (p *InMemoryStoreProvider) StoreValue(storeName, key string, value interface{}) {
	if _, ok := p.stores[storeName]; !ok {
		p.stores[storeName] = &storeData{data: make(map[string]interface{})}
	}
	key = applyKeyPrefix(key)
	p.stores[storeName].data[key] = value
}

func (p *InMemoryStoreProvider) GetAllValues(storeName, keyPrefix string) map[string]interface{} {
	store, ok := p.stores[storeName]
	if !ok {
		return nil
	}
	result := make(map[string]interface{})
	keyPrefix = applyKeyPrefix(keyPrefix)
	for k, v := range store.data {
		if strings.HasPrefix(k, keyPrefix) {
			deprefixedKey := removeKeyPrefix(k)
			result[deprefixedKey] = v
		}
	}
	return result
}

func (p *InMemoryStoreProvider) DeleteValue(storeName, key string) {
	store, ok := p.stores[storeName]
	if ok {
		key = applyKeyPrefix(key)
		delete(store.data, key)
	}
}

func (p *InMemoryStoreProvider) DeleteStore(storeName string) {
	delete(p.stores, storeName)
}
