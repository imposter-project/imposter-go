//go:build integration

package store

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testTableName = "imposter-store-test"

func startDynamoDBContainer(t *testing.T) testcontainers.Container {
	t.Helper()
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "amazon/dynamodb-local:latest",
			ExposedPorts: []string{"8000/tcp"},
			Cmd:          []string{"-jar", "DynamoDBLocal.jar", "-inMemory"},
			WaitingFor:   wait.ForListeningPort("8000/tcp"),
		},
		Started: true,
	})
	require.NoError(t, err, "failed to start DynamoDB Local container")
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(ctx))
	})
	return container
}

func createTestTable(t *testing.T, endpoint string) {
	t.Helper()
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Endpoint:    aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials("dummy", "dummy", ""),
	}))
	ddb := dynamodb.New(sess)

	_, err := ddb.CreateTable(&dynamodb.CreateTableInput{
		TableName: aws.String(testTableName),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{AttributeName: aws.String("StoreName"), AttributeType: aws.String("S")},
			{AttributeName: aws.String("Key"), AttributeType: aws.String("S")},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{AttributeName: aws.String("StoreName"), KeyType: aws.String("HASH")},
			{AttributeName: aws.String("Key"), KeyType: aws.String("RANGE")},
		},
		BillingMode: aws.String("PAY_PER_REQUEST"),
	})
	require.NoError(t, err, "failed to create test table")
}

func setupDynamoDBIntegration(t *testing.T, container testcontainers.Container) *DynamoDBStoreProvider {
	t.Helper()
	ctx := context.Background()
	endpoint, err := container.Endpoint(ctx, "http")
	require.NoError(t, err)

	createTestTable(t, endpoint)

	t.Setenv("AWS_ENDPOINT_URL", endpoint)
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "dummy")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
	t.Setenv("IMPOSTER_STORE_DYNAMODB_TABLE", testTableName)
	t.Setenv("IMPOSTER_STORE_DYNAMODB_REGION", "us-east-1")

	provider := &DynamoDBStoreProvider{}
	provider.InitStores()
	return provider
}

func TestDynamoDBIntegration_BasicCRUD(t *testing.T) {
	container := startDynamoDBContainer(t)
	provider := setupDynamoDBIntegration(t, container)

	t.Run("StoreAndGetValue", func(t *testing.T) {
		provider.StoreValue("test", "key1", "value1")
		val, found := provider.GetValue("test", "key1")
		assert.True(t, found)
		assert.Equal(t, "value1", val)
	})

	t.Run("GetNonExistentValue", func(t *testing.T) {
		_, found := provider.GetValue("test", "nonexistent")
		assert.False(t, found)
	})

	t.Run("OverwriteValue", func(t *testing.T) {
		provider.StoreValue("test", "overwrite", "first")
		provider.StoreValue("test", "overwrite", "second")
		val, found := provider.GetValue("test", "overwrite")
		assert.True(t, found)
		assert.Equal(t, "second", val)
	})

	t.Run("DeleteValue", func(t *testing.T) {
		provider.StoreValue("test", "todelete", "value")
		provider.DeleteValue("test", "todelete")
		_, found := provider.GetValue("test", "todelete")
		assert.False(t, found)
	})

	t.Run("DeleteNonExistentValue", func(t *testing.T) {
		provider.DeleteValue("test", "never-existed")
	})
}

func TestDynamoDBIntegration_ComplexValues(t *testing.T) {
	container := startDynamoDBContainer(t)
	provider := setupDynamoDBIntegration(t, container)

	t.Run("MapValue", func(t *testing.T) {
		v := map[string]interface{}{
			"name": "test",
			"age":  float64(30),
			"nested": map[string]interface{}{
				"key": "value",
			},
		}
		provider.StoreValue("test", "complex", v)
		val, found := provider.GetValue("test", "complex")
		require.True(t, found)
		m, ok := val.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test", m["name"])
		assert.Equal(t, float64(30), m["age"])
		nested, ok := m["nested"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", nested["key"])
	})

	t.Run("ArrayValue", func(t *testing.T) {
		v := []interface{}{"a", "b", "c"}
		provider.StoreValue("test", "arr", v)
		val, found := provider.GetValue("test", "arr")
		require.True(t, found)
		arr, ok := val.([]interface{})
		require.True(t, ok)
		assert.Equal(t, []interface{}{"a", "b", "c"}, arr)
	})

	t.Run("NumericValue", func(t *testing.T) {
		provider.StoreValue("test", "num", 42.5)
		val, found := provider.GetValue("test", "num")
		require.True(t, found)
		assert.Equal(t, 42.5, val)
	})

	t.Run("BooleanValue", func(t *testing.T) {
		provider.StoreValue("test", "flag", true)
		val, found := provider.GetValue("test", "flag")
		require.True(t, found)
		assert.Equal(t, true, val)
	})

	t.Run("NullValue", func(t *testing.T) {
		provider.StoreValue("test", "null", nil)
		val, found := provider.GetValue("test", "null")
		require.True(t, found)
		assert.Nil(t, val)
	})
}

func TestDynamoDBIntegration_GetAllValues(t *testing.T) {
	container := startDynamoDBContainer(t)
	provider := setupDynamoDBIntegration(t, container)

	provider.StoreValue("test", "prefix.key1", "value1")
	provider.StoreValue("test", "prefix.key2", "value2")
	provider.StoreValue("test", "other.key3", "value3")

	t.Run("WithMatchingPrefix", func(t *testing.T) {
		values := provider.GetAllValues("test", "prefix")
		assert.Len(t, values, 2)
		assert.Equal(t, "value1", values["key1"])
		assert.Equal(t, "value2", values["key2"])
	})

	t.Run("WithNoMatchingPrefix", func(t *testing.T) {
		values := provider.GetAllValues("test", "nomatch")
		assert.Empty(t, values)
	})

	t.Run("AcrossStores", func(t *testing.T) {
		provider.StoreValue("store-a", "shared.k", "from-a")
		provider.StoreValue("store-b", "shared.k", "from-b")
		valA := provider.GetAllValues("store-a", "shared")
		valB := provider.GetAllValues("store-b", "shared")
		assert.Equal(t, "from-a", valA["k"])
		assert.Equal(t, "from-b", valB["k"])
	})
}

func TestDynamoDBIntegration_TTL(t *testing.T) {
	container := startDynamoDBContainer(t)
	provider := setupDynamoDBIntegration(t, container)

	t.Setenv("IMPOSTER_STORE_DYNAMODB_TTL", "3600")

	provider.StoreValue("test", "with-ttl", "value")

	result, err := provider.ddb.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(provider.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"StoreName": {S: aws.String("test")},
			"Key":       {S: aws.String("with-ttl")},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result.Item["ttl"], "TTL attribute should be set")
}

func TestDynamoDBIntegration_CustomTTLAttribute(t *testing.T) {
	container := startDynamoDBContainer(t)
	provider := setupDynamoDBIntegration(t, container)

	t.Setenv("IMPOSTER_STORE_DYNAMODB_TTL", "3600")
	t.Setenv("IMPOSTER_STORE_DYNAMODB_TTL_ATTRIBUTE", "expiresAt")

	provider.StoreValue("test", "custom-ttl", "value")

	result, err := provider.ddb.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(provider.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"StoreName": {S: aws.String("test")},
			"Key":       {S: aws.String("custom-ttl")},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result.Item["expiresAt"], "custom TTL attribute should be set")
	assert.Nil(t, result.Item["ttl"], "default TTL attribute should not be set")
}

func TestDynamoDBIntegration_AtomicOperations(t *testing.T) {
	container := startDynamoDBContainer(t)
	provider := setupDynamoDBIntegration(t, container)

	t.Run("Increment", func(t *testing.T) {
		val, err := provider.AtomicIncrement("test", "counter", 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)

		val, err = provider.AtomicIncrement("test", "counter", 5)
		require.NoError(t, err)
		assert.Equal(t, int64(6), val)
	})

	t.Run("Decrement", func(t *testing.T) {
		val, err := provider.AtomicIncrement("test", "dec-counter", 10)
		require.NoError(t, err)
		assert.Equal(t, int64(10), val)

		val, err = provider.AtomicDecrement("test", "dec-counter", 3)
		require.NoError(t, err)
		assert.Equal(t, int64(7), val)
	})

	t.Run("ConcurrentIncrements", func(t *testing.T) {
		const goroutines = 50
		const incrementsPerGoroutine = 20
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < incrementsPerGoroutine; j++ {
					_, err := provider.AtomicIncrement("test", "concurrent", 1)
					assert.NoError(t, err)
				}
			}()
		}
		wg.Wait()

		val, err := provider.AtomicIncrement("test", "concurrent", 0)
		require.NoError(t, err)
		assert.Equal(t, int64(goroutines*incrementsPerGoroutine), val)
	})
}

func TestDynamoDBIntegration_KeyPrefix(t *testing.T) {
	container := startDynamoDBContainer(t)

	t.Setenv("IMPOSTER_STORE_KEY_PREFIX", "myprefix")
	provider := setupDynamoDBIntegration(t, container)

	provider.StoreValue("test", "key1", "value1")
	val, found := provider.GetValue("test", "key1")
	assert.True(t, found)
	assert.Equal(t, "value1", val)

	result, err := provider.ddb.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(provider.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"StoreName": {S: aws.String("test")},
			"Key":       {S: aws.String("myprefix.key1")},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, result.Item, "key should be stored with prefix in DynamoDB")

	os.Unsetenv("IMPOSTER_STORE_KEY_PREFIX")
}

func TestDynamoDBIntegration_StoreIsolation(t *testing.T) {
	container := startDynamoDBContainer(t)
	provider := setupDynamoDBIntegration(t, container)

	provider.StoreValue("store1", "key", "value-from-store1")
	provider.StoreValue("store2", "key", "value-from-store2")

	val1, found := provider.GetValue("store1", "key")
	assert.True(t, found)
	assert.Equal(t, "value-from-store1", val1)

	val2, found := provider.GetValue("store2", "key")
	assert.True(t, found)
	assert.Equal(t, "value-from-store2", val2)
}

func TestDynamoDBIntegration_LargeDataSet(t *testing.T) {
	container := startDynamoDBContainer(t)
	provider := setupDynamoDBIntegration(t, container)

	const count = 100
	for i := 0; i < count; i++ {
		provider.StoreValue("bulk", fmt.Sprintf("item.%03d", i), fmt.Sprintf("value-%d", i))
	}

	values := provider.GetAllValues("bulk", "item")
	assert.Len(t, values, count)

	for i := 0; i < count; i++ {
		key := fmt.Sprintf("%03d", i)
		assert.Equal(t, fmt.Sprintf("value-%d", i), values[key])
	}
}
