package store

import (
	"strings"
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
