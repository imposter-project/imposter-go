package store

import (
	"strings"
)

type RequestStoreProvider struct {
	data map[string]interface{}
}

func (p *RequestStoreProvider) InitStores() {
	// no-op
}

func (p *RequestStoreProvider) GetValue(storeName, key string) (interface{}, bool) {
	if p.data == nil {
		return nil, false
	}
	key = applyKeyPrefix(key)
	val, found := p.data[key]
	return val, found
}

func (p *RequestStoreProvider) StoreValue(storeName, key string, value interface{}) {
	if p.data == nil {
		p.data = make(map[string]interface{})
	}
	key = applyKeyPrefix(key)
	p.data[key] = value
}

func (p *RequestStoreProvider) GetAllValues(storeName, keyPrefix string) map[string]interface{} {
	if p.data == nil {
		return nil
	}

	result := make(map[string]interface{})
	prefixToMatch := applyKeyPrefix(keyPrefix)

	for k, v := range p.data {
		if strings.HasPrefix(k, prefixToMatch) {
			// Remove the prefix we're searching for from the key
			key := strings.TrimPrefix(k, prefixToMatch)
			// If there's a dot at the start (from the prefix), remove it
			key = strings.TrimPrefix(key, ".")
			result[key] = v
		}
	}
	return result
}

func (p *RequestStoreProvider) DeleteValue(storeName, key string) {
	if p.data != nil {
		key = applyKeyPrefix(key)
		delete(p.data, key)
	}
}

func (p *RequestStoreProvider) DeleteStore(storeName string) {
	p.data = nil
}

// NewRequestStore creates a new request store, backed by a map
func NewRequestStore() *Store {
	// unlike other store implementations, a new provider is created for each request store,
	// as the request store always has the name "request", but is not shared across requests
	return &Store{
		name:     "request",
		provider: &RequestStoreProvider{},
	}
}
