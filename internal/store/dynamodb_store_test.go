package store

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func setupDynamoDBTest(t *testing.T) *DynamoDBStoreProvider {
	// Skip if DynamoDB configuration is not available
	tableName := os.Getenv("IMPOSTER_STORE_DYNAMODB_TABLE")
	if tableName == "" {
		t.Skip("Skipping DynamoDB tests: IMPOSTER_STORE_DYNAMODB_TABLE not set")
	}

	provider := &DynamoDBStoreProvider{}
	provider.InitStores()

	// Clear test data
	clearDynamoDBTable(t, provider)

	return provider
}

func clearDynamoDBTable(t *testing.T, provider *DynamoDBStoreProvider) {
	// Scan for all items
	input := &dynamodb.ScanInput{
		TableName: aws.String(provider.tableName),
	}

	result, err := provider.ddb.Scan(input)
	if err != nil {
		t.Fatalf("Failed to scan table: %v", err)
	}

	// Delete all items
	for _, item := range result.Items {
		_, err := provider.ddb.DeleteItem(&dynamodb.DeleteItemInput{
			TableName: aws.String(provider.tableName),
			Key: map[string]*dynamodb.AttributeValue{
				"StoreName": item["StoreName"],
				"Key":       item["Key"],
			},
		})
		if err != nil {
			t.Fatalf("Failed to delete item: %v", err)
		}
	}
}

func TestDynamoDBStore(t *testing.T) {
	provider := setupDynamoDBTest(t)

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
		provider.StoreValue("test", "prefix.key1", "value1")
		provider.StoreValue("test", "prefix.key2", "value2")
		provider.StoreValue("test", "other.key3", "value3")

		values := provider.GetAllValues("test", "prefix")
		if len(values) != 2 {
			t.Errorf("Expected 2 values but got %d", len(values))
		}
		if values["key1"] != "value1" || values["key2"] != "value2" {
			t.Error("Got unexpected values")
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

	t.Run("TTL", func(t *testing.T) {
		// Set a TTL for testing
		os.Setenv("IMPOSTER_STORE_DYNAMODB_TTL", "1")
		defer os.Unsetenv("IMPOSTER_STORE_DYNAMODB_TTL")

		provider.StoreValue("test", "expiring", "value")

		// Note: We can't effectively test TTL expiration in DynamoDB
		// as it's eventually consistent and can take up to 48 hours
		// Instead, we'll verify the TTL attribute was set
		input := &dynamodb.GetItemInput{
			TableName: aws.String(provider.tableName),
			Key: map[string]*dynamodb.AttributeValue{
				"StoreName": {S: aws.String("test")},
				"Key":       {S: aws.String("expiring")},
			},
		}

		result, err := provider.ddb.GetItem(input)
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}

		if result.Item["ttl"] == nil {
			t.Error("TTL attribute not set")
		}
	})
}

func TestDynamoDBConnection(t *testing.T) {
	t.Run("InvalidCredentials", func(t *testing.T) {
		// Save current AWS config
		origRegion := os.Getenv("AWS_REGION")
		origKey := os.Getenv("AWS_ACCESS_KEY_ID")
		origSecret := os.Getenv("AWS_SECRET_ACCESS_KEY")

		// Set invalid credentials
		os.Setenv("AWS_REGION", "invalid-region")
		os.Setenv("AWS_ACCESS_KEY_ID", "invalid-key")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "invalid-secret")
		defer func() {
			// Restore AWS config
			os.Setenv("AWS_REGION", origRegion)
			os.Setenv("AWS_ACCESS_KEY_ID", origKey)
			os.Setenv("AWS_SECRET_ACCESS_KEY", origSecret)
		}()

		provider := &DynamoDBStoreProvider{}
		provider.InitStores()

		// Operations should fail gracefully
		provider.StoreValue("test", "key", "value")
		_, found := provider.GetValue("test", "key")
		if found {
			t.Error("Expected operation to fail with invalid credentials")
		}
	})
}

func TestDynamoDBTTLAttribute(t *testing.T) {
	t.Run("CustomTTLAttribute", func(t *testing.T) {
		os.Setenv("IMPOSTER_STORE_DYNAMODB_TTL_ATTRIBUTE", "customTTL")
		defer os.Unsetenv("IMPOSTER_STORE_DYNAMODB_TTL_ATTRIBUTE")

		if getTTLAttributeName() != "customTTL" {
			t.Error("Expected custom TTL attribute name")
		}
	})

	t.Run("DefaultTTLAttribute", func(t *testing.T) {
		os.Unsetenv("IMPOSTER_STORE_DYNAMODB_TTL_ATTRIBUTE")
		if getTTLAttributeName() != "ttl" {
			t.Error("Expected default TTL attribute name")
		}
	})
}
