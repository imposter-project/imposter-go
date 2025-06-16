package store

type RequestStoreProvider struct {
	inMemoryProvider *InMemoryStoreProvider
}

func (p *RequestStoreProvider) InitStores() {
	p.inMemoryProvider = &InMemoryStoreProvider{}
	p.inMemoryProvider.InitStores()
}

func (p *RequestStoreProvider) GetValue(storeName, key string) (interface{}, bool) {
	return p.inMemoryProvider.GetValue(storeName, key)
}

func (p *RequestStoreProvider) StoreValue(storeName, key string, value interface{}) {
	p.inMemoryProvider.StoreValue(storeName, key, value)
}

func (p *RequestStoreProvider) GetAllValues(storeName, keyPrefix string) map[string]interface{} {
	return p.inMemoryProvider.GetAllValues(storeName, keyPrefix)
}

func (p *RequestStoreProvider) DeleteValue(storeName, key string) {
	p.inMemoryProvider.DeleteValue(storeName, key)
}

func (p *RequestStoreProvider) DeleteStore(storeName string) {
	p.inMemoryProvider.DeleteStore(storeName)
}

func (p *RequestStoreProvider) AtomicIncrement(storeName, key string, delta int64) (int64, error) {
	return p.inMemoryProvider.AtomicIncrement(storeName, key, delta)
}

func (p *RequestStoreProvider) AtomicDecrement(storeName, key string, delta int64) (int64, error) {
	return p.inMemoryProvider.AtomicDecrement(storeName, key, delta)
}

// NewRequestStore creates a new request store, backed by a map
func NewRequestStore() *Store {
	// unlike other store implementations, a new provider is created for each request store,
	// as the request store always has the name "request", but is not shared across requests
	provider := &RequestStoreProvider{}
	provider.InitStores()
	return &Store{
		name:     "request",
		provider: provider,
	}
}
