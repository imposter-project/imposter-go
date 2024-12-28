package store

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gatehill/imposter-go/internal/config"
)

type DynamoDBStoreProvider struct {
	ddb       *dynamodb.DynamoDB
	tableName string
}

func (p *DynamoDBStoreProvider) InitStores() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	}))
	p.ddb = dynamodb.New(sess)
	p.tableName = os.Getenv("IMPOSTER_DYNAMODB_TABLE")
}

func (p *DynamoDBStoreProvider) PreloadStores(configDir string, configs []config.Config) {
	// No-op for now
}

func (p *DynamoDBStoreProvider) GetValue(storeName, key string) (interface{}, bool) {
	key = applyKeyPrefix(key)
	result, err := p.ddb.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(p.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"StoreName": {S: aws.String(storeName)},
			"Key":       {S: aws.String(key)},
		},
	})
	if err != nil {
		fmt.Printf("Failed to get item: %v\n", err)
		return nil, false
	}
	if result.Item == nil {
		return nil, false
	}
	var value interface{}
	if err := json.Unmarshal([]byte(*result.Item["Value"].S), &value); err != nil {
		fmt.Printf("Failed to unmarshal value: %v\n", err)
		return nil, false
	}
	return value, true
}

func (p *DynamoDBStoreProvider) StoreValue(storeName, key string, value interface{}) {
	key = applyKeyPrefix(key)
	valueBytes, err := json.Marshal(value)
	if err != nil {
		fmt.Printf("Failed to marshal value: %v\n", err)
		return
	}
	_, err = p.ddb.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(p.tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"StoreName": {S: aws.String(storeName)},
			"Key":       {S: aws.String(key)},
			"Value":     {S: aws.String(string(valueBytes))},
		},
	})
	if err != nil {
		fmt.Printf("Failed to put item: %v\n", err)
	}
}

func (p *DynamoDBStoreProvider) GetAllValues(storeName, keyPrefix string) map[string]interface{} {
	keyPrefix = applyKeyPrefix(keyPrefix)
	result, err := p.ddb.Query(&dynamodb.QueryInput{
		TableName:              aws.String(p.tableName),
		KeyConditionExpression: aws.String("StoreName = :storeName AND begins_with(#k, :keyPrefix)"),
		ExpressionAttributeNames: map[string]*string{
			"#k": aws.String("Key"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":storeName": {S: aws.String(storeName)},
			":keyPrefix": {S: aws.String(keyPrefix)},
		},
	})
	if err != nil {
		fmt.Printf("Failed to query items: %v\n", err)
		return nil
	}
	items := make(map[string]interface{})
	for _, item := range result.Items {
		var value interface{}
		if err := json.Unmarshal([]byte(*item["Value"].S), &value); err != nil {
			fmt.Printf("Failed to unmarshal value: %v\n", err)
			continue
		}
		deprefixedKey := removeKeyPrefix(*item["Key"].S)
		items[deprefixedKey] = value
	}
	return items
}

func (p *DynamoDBStoreProvider) DeleteValue(storeName, key string) {
	key = applyKeyPrefix(key)
	_, err := p.ddb.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(p.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"StoreName": {S: aws.String(storeName)},
			"Key":       {S: aws.String(key)},
		},
	})
	if err != nil {
		fmt.Printf("Failed to delete item: %v\n", err)
	}
}

func (p *DynamoDBStoreProvider) DeleteStore(storeName string) {
	// No-op for now
}
