package store

import (
	"testing"
)

func setupRequestTest(t *testing.T) *RequestStoreProvider {
	provider := &RequestStoreProvider{}
	provider.InitStores()
	return provider
}

func TestRequestStoreProvider(t *testing.T) {
	provider := setupRequestTest(t)

	t.Run("StoreAndGetValue", func(t *testing.T) {
		provider.StoreValue("test", "key1", "value1")
		val, found := provider.GetValue("test", "key1")
		if !found {
			t.Error("Expected to find value but got not found")
		}
		if val != "value1" {
			t.Errorf("Expected value1 but got %v", val)
		}
	})

	t.Run("GetNonExistentValue", func(t *testing.T) {
		_, found := provider.GetValue("test", "nonexistent")
		if found {
			t.Error("Expected not found for nonexistent key")
		}
	})

	t.Run("GetAllValues", func(t *testing.T) {
		// Clear any previous test data
		provider.DeleteStore("test")

		// Store some test data
		provider.StoreValue("test", "prefix.key1", "value1")
		provider.StoreValue("test", "prefix.key2", "value2")
		provider.StoreValue("test", "other.key3", "value3")

		// Get all values with prefix
		values := provider.GetAllValues("test", "prefix")

		// Debug output
		t.Logf("Stored values: %v", provider.data)
		t.Logf("Retrieved values: %v", values)

		// Check the number of values
		if len(values) != 2 {
			t.Errorf("Expected 2 values but got %d. Values: %v", len(values), values)
		}

		// Check each value individually
		expectedValues := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		for key, expectedValue := range expectedValues {
			actualValue, ok := values[key]
			if !ok {
				t.Errorf("Expected to find key %q but it was missing. All values: %v", key, values)
				continue
			}
			if actualValue != expectedValue {
				t.Errorf("For key %q expected value %q but got %q", key, expectedValue, actualValue)
			}
		}

		// Verify other.key3 is not included
		if _, ok := values["key3"]; ok {
			t.Errorf("key3 should not be included in the results. All values: %v", values)
		}
	})

	t.Run("DeleteValue", func(t *testing.T) {
		provider.StoreValue("test", "key1", "value1")
		provider.DeleteValue("test", "key1")
		_, found := provider.GetValue("test", "key1")
		if found {
			t.Error("Value should have been deleted")
		}
	})

	t.Run("DeleteStore", func(t *testing.T) {
		provider.StoreValue("test", "key1", "value1")
		provider.DeleteStore("test")
		_, found := provider.GetValue("test", "key1")
		if found {
			t.Error("Store should have been deleted")
		}
	})

	t.Run("StoreComplexValue", func(t *testing.T) {
		complexValue := map[string]interface{}{
			"name": "test",
			"age":  30,
			"nested": map[string]interface{}{
				"key": "value",
			},
		}
		provider.StoreValue("test", "complex", complexValue)
		val, found := provider.GetValue("test", "complex")
		if !found {
			t.Error("Expected to find complex value")
		}
		mapVal, ok := val.(map[string]interface{})
		if !ok {
			t.Error("Expected map type for complex value")
		}
		if mapVal["name"] != "test" || mapVal["age"] != 30 {
			t.Error("Complex value not stored correctly")
		}
	})
}
