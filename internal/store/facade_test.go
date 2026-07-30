package store

import (
	"testing"
)

// TestStoreFacade exercises the Store handle methods that delegate to the
// configured provider, using the in-memory provider as the backing store.
func TestStoreFacade(t *testing.T) {
	storeProvider = &InMemoryStoreProvider{}
	storeProvider.InitStores()

	t.Run("StoreAndGetValue", func(t *testing.T) {
		s := Open("facade", nil)
		s.StoreValue("key", "value")
		val, found := s.GetValue("key")
		if !found || val != "value" {
			t.Errorf("expected to retrieve 'value', got %v (found=%v)", val, found)
		}
	})

	t.Run("DeleteValue", func(t *testing.T) {
		s := Open("facade", nil)
		s.StoreValue("temp", "value")
		s.DeleteValue("temp")
		if _, found := s.GetValue("temp"); found {
			t.Error("expected value to be deleted via the facade")
		}
	})

	t.Run("GetAllValues", func(t *testing.T) {
		s := Open("facade-all", nil)
		s.StoreValue("p.a", "1")
		s.StoreValue("p.b", "2")
		s.StoreValue("q.c", "3")
		values := s.GetAllValues("p.")
		if len(values) != 2 {
			t.Errorf("expected 2 values with prefix 'p.', got %d: %v", len(values), values)
		}
	})

	t.Run("AtomicIncrementAndDecrement", func(t *testing.T) {
		s := Open("facade-counter", nil)
		if got, err := s.AtomicIncrement("n", 5); err != nil || got != 5 {
			t.Errorf("increment: expected 5, got %d (err=%v)", got, err)
		}
		if got, err := s.AtomicDecrement("n", 2); err != nil || got != 3 {
			t.Errorf("decrement: expected 3, got %d (err=%v)", got, err)
		}
	})
}

// TestDeleteStore verifies the package-level DeleteStore removes an entire named
// store from the global provider.
func TestDeleteStore(t *testing.T) {
	storeProvider = &InMemoryStoreProvider{}
	storeProvider.InitStores()

	s := Open("disposable", nil)
	s.StoreValue("key", "value")
	DeleteStore("disposable")

	if _, found := s.GetValue("key"); found {
		t.Error("expected the store to be removed by DeleteStore")
	}
}

// TestGetStoreProvider verifies the accessor returns the configured global
// provider instance.
func TestGetStoreProvider(t *testing.T) {
	provider := &InMemoryStoreProvider{}
	provider.InitStores()
	storeProvider = provider

	if GetStoreProvider() != provider {
		t.Error("expected GetStoreProvider to return the configured provider")
	}
}

// TestOpenRequestStore verifies that opening the reserved "request" store name
// returns the supplied per-request store rather than the global provider.
func TestOpenRequestStore(t *testing.T) {
	storeProvider = &InMemoryStoreProvider{}
	storeProvider.InitStores()

	reqStore := NewRequestStore()
	opened := Open("request", reqStore)
	if opened != reqStore {
		t.Error("expected Open(\"request\", ...) to return the supplied request store")
	}
}

// TestNewRequestStore_Isolation verifies each request store is independent, so
// values written during one request are not visible to another. This isolation
// is the reason request stores use a fresh provider per request.
func TestNewRequestStore_Isolation(t *testing.T) {
	first := NewRequestStore()
	second := NewRequestStore()

	first.StoreValue("token", "abc")

	if _, found := second.GetValue("token"); found {
		t.Error("expected request stores to be isolated from one another")
	}

	val, found := first.GetValue("token")
	if !found || val != "abc" {
		t.Errorf("expected the originating request store to retain its value, got %v (found=%v)", val, found)
	}
}
