package store

import (
	"fmt"
	"os"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
)

func TestPreloadStores(t *testing.T) {
	// Setup temporary test directory
	tmpDir := t.TempDir()

	// Create a test JSON file
	testData := `{"key1": "value1", "key2": "value2"}`
	err := os.WriteFile(tmpDir+"/test.json", []byte(testData), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Setup test config
	configs := []config.Config{
		{
			System: &config.System{
				Stores: map[string]config.StoreDefinition{
					"fileStore": {
						PreloadFile: "test.json",
					},
					"inlineStore": {
						PreloadData: map[string]interface{}{
							"key3": "value3",
							"key4": "value4",
						},
					},
				},
			},
		},
	}

	// Initialize store provider
	storeProvider = &InMemoryStoreProvider{}
	storeProvider.InitStores()

	// Test preloading
	PreloadStores(tmpDir, configs)

	t.Run("PreloadedFileStore", func(t *testing.T) {
		val, found := GetValue("fileStore", "key1")
		if !found || val != "value1" {
			t.Error("File preload failed")
		}
		val, found = GetValue("fileStore", "key2")
		if !found || val != "value2" {
			t.Error("File preload failed")
		}
	})

	t.Run("PreloadedInlineStore", func(t *testing.T) {
		val, found := GetValue("inlineStore", "key3")
		if !found || val != "value3" {
			t.Error("Inline preload failed")
		}
		val, found = GetValue("inlineStore", "key4")
		if !found || val != "value4" {
			t.Error("Inline preload failed")
		}
	})
}

func TestStoreKeyPrefix(t *testing.T) {
	// Setup
	storeProvider = &InMemoryStoreProvider{}
	storeProvider.InitStores()

	t.Run("WithoutPrefix", func(t *testing.T) {
		os.Unsetenv("IMPOSTER_STORE_KEY_PREFIX")
		StoreValue("test", "key1", "value1")
		val, found := GetValue("test", "key1")
		if !found || val != "value1" {
			t.Error("Store without prefix failed")
		}
	})

	t.Run("WithPrefix", func(t *testing.T) {
		os.Setenv("IMPOSTER_STORE_KEY_PREFIX", "prefix")
		defer os.Unsetenv("IMPOSTER_STORE_KEY_PREFIX")

		StoreValue("test", "key1", "value1")
		val, found := GetValue("test", "key1")
		if !found || val != "value1" {
			t.Error("Store with prefix failed")
		}

		// Test GetAllValues with prefix
		values := GetAllValues("test", "key")
		if len(values) != 1 {
			t.Error("GetAllValues with prefix failed")
		}
	})
}

func TestStoreProviderSelection(t *testing.T) {
	// Save current provider
	originalProvider := storeProvider
	defer func() {
		storeProvider = originalProvider
	}()

	t.Run("DefaultProvider", func(t *testing.T) {
		os.Unsetenv("IMPOSTER_STORE_DRIVER")
		InitStoreProvider()
		_, ok := storeProvider.(*InMemoryStoreProvider)
		if !ok {
			t.Error("Expected InMemoryStoreProvider as default")
		}
	})

	t.Run("DynamoDBProvider", func(t *testing.T) {
		// Skip if AWS credentials are not set
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
			t.Skip("Skipping DynamoDB test: AWS credentials not set")
		}

		os.Setenv("IMPOSTER_STORE_DRIVER", "store-dynamodb")
		defer os.Unsetenv("IMPOSTER_STORE_DRIVER")
		InitStoreProvider()
		_, ok := storeProvider.(*DynamoDBStoreProvider)
		if !ok {
			t.Error("Expected DynamoDBStoreProvider")
		}
	})

	t.Run("RedisProvider", func(t *testing.T) {
		// Skip if Redis address is not set
		if os.Getenv("REDIS_ADDR") == "" {
			t.Skip("Skipping Redis test: REDIS_ADDR not set")
		}

		os.Setenv("IMPOSTER_STORE_DRIVER", "store-redis")
		defer os.Unsetenv("IMPOSTER_STORE_DRIVER")
		InitStoreProvider()
		_, ok := storeProvider.(*RedisStoreProvider)
		if !ok {
			t.Error("Expected RedisStoreProvider")
		}
	})
}

func TestStore_ThreadSafety(t *testing.T) {
	// Setup
	storeProvider = &InMemoryStoreProvider{}
	storeProvider.InitStores()

	t.Run("ConcurrentAccess", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				// Write
				StoreValue("test", fmt.Sprintf("key%d", id), fmt.Sprintf("value%d", id))
				// Read
				_, _ = GetValue("test", fmt.Sprintf("key%d", id))
				// Read all
				_ = GetAllValues("test", "key")
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
