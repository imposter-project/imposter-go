package store

import (
	"strings"
	"sync"
)

type InMemoryStoreProvider struct {
	stores map[string]*storeData
	mu     sync.RWMutex
}

type storeData struct {
	data map[string]interface{}
}

func (p *InMemoryStoreProvider) InitStores() {
	p.stores = make(map[string]*storeData)
}

func (p *InMemoryStoreProvider) GetValue(storeName, key string) (interface{}, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	store, ok := p.stores[storeName]
	if !ok {
		return nil, false
	}
	key = applyKeyPrefix(key)
	val, found := store.data[key]
	return val, found
}

func (p *InMemoryStoreProvider) StoreValue(storeName, key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.stores[storeName]; !ok {
		p.stores[storeName] = &storeData{data: make(map[string]interface{})}
	}
	key = applyKeyPrefix(key)
	p.stores[storeName].data[key] = value
}

func (p *InMemoryStoreProvider) GetAllValues(storeName, keyPrefix string) map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]interface{})

	store, ok := p.stores[storeName]
	if !ok {
		return result
	}

	prefixToMatch := applyKeyPrefix(keyPrefix)

	for k, v := range store.data {
		if strings.HasPrefix(k, prefixToMatch) {
			// If there's a dot at the start (from the prefix), remove it
			key := strings.TrimPrefix(k, ".")
			result[key] = v
		}
	}
	return result
}

func (p *InMemoryStoreProvider) DeleteValue(storeName, key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	store, ok := p.stores[storeName]
	if ok {
		key = applyKeyPrefix(key)
		delete(store.data, key)
	}
}

func (p *InMemoryStoreProvider) DeleteStore(storeName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.stores, storeName)
}
