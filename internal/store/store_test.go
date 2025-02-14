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

	// Initialise store provider
	storeProvider = &InMemoryStoreProvider{}
	storeProvider.InitStores()

	// Test preloading
	PreloadStores(tmpDir, configs)

	t.Run("PreloadedFileStore", func(t *testing.T) {
		s := Open("fileStore", nil)
		val, found := s.GetValue("key1")
		if !found || val != "value1" {
			t.Error("File preload failed")
		}
		val, found = s.GetValue("key2")
		if !found || val != "value2" {
			t.Error("File preload failed")
		}
	})

	t.Run("PreloadedInlineStore", func(t *testing.T) {
		s := Open("inlineStore", nil)
		val, found := s.GetValue("key3")
		if !found || val != "value3" {
			t.Error("Inline preload failed")
		}
		val, found = s.GetValue("key4")
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
		s := Open("test", nil)
		s.StoreValue("key1", "value1")
		val, found := s.GetValue("key1")
		if !found || val != "value1" {
			t.Error("Store without prefix failed")
		}
	})

	t.Run("WithPrefix", func(t *testing.T) {
		os.Setenv("IMPOSTER_STORE_KEY_PREFIX", "prefix")
		defer os.Unsetenv("IMPOSTER_STORE_KEY_PREFIX")

		s := Open("test", nil)
		s.StoreValue("key1", "value1")
		val, found := s.GetValue("key1")
		if !found || val != "value1" {
			t.Error("Store with prefix failed")
		}

		// Test GetAllValues with prefix
		values := s.GetAllValues("key")
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
				s := Open("test", nil)
				// Write
				s.StoreValue(fmt.Sprintf("key%d", id), fmt.Sprintf("value%d", id))
				// Read
				_, _ = s.GetValue(fmt.Sprintf("key%d", id))
				// Read all
				_ = s.GetAllValues("key")
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
