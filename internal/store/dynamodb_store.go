package store

import (
	"encoding/json"
	"fmt"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type DynamoDBStoreProvider struct {
	ddb       *dynamodb.DynamoDB
	tableName string
}

func (p *DynamoDBStoreProvider) InitStores() {
	region := os.Getenv("IMPOSTER_STORE_DYNAMODB_REGION")
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	p.ddb = dynamodb.New(sess)
	p.tableName = os.Getenv("IMPOSTER_STORE_DYNAMODB_TABLE")
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
		logger.Errorf("failed to get item: %v", err)
		return nil, false
	}
	if result.Item == nil {
		return nil, false
	}
	var value interface{}
	if err := json.Unmarshal([]byte(*result.Item["Value"].S), &value); err != nil {
		logger.Errorf("failed to unmarshal value: %v", err)
		return nil, false
	}
	return value, true
}

func (p *DynamoDBStoreProvider) StoreValue(storeName, key string, value interface{}) {
	key = applyKeyPrefix(key)
	valueBytes, err := json.Marshal(value)
	if err != nil {
		logger.Errorf("failed to marshal value: %v", err)
		return
	}

	item := map[string]*dynamodb.AttributeValue{
		"StoreName": {S: aws.String(storeName)},
		"Key":       {S: aws.String(key)},
		"Value":     {S: aws.String(string(valueBytes))},
	}

	ttl := getTTL()
	if ttl > 0 {
		item[getTTLAttributeName()] = &dynamodb.AttributeValue{N: aws.String(fmt.Sprintf("%d", time.Now().Add(time.Duration(ttl)*time.Second).Unix()))}
	}

	_, err = p.ddb.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(p.tableName),
		Item:      item,
	})
	if err != nil {
		logger.Errorf("failed to put item: %v", err)
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
		logger.Errorf("failed to query items: %v", err)
		return nil
	}
	items := make(map[string]interface{})
	for _, item := range result.Items {
		var value interface{}
		if err := json.Unmarshal([]byte(*item["Value"].S), &value); err != nil {
			logger.Errorf("failed to unmarshal value: %v", err)
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
		logger.Errorf("failed to delete item: %v", err)
	}
}

func (p *DynamoDBStoreProvider) DeleteStore(storeName string) {
	// No-op for now
}

func getTTL() int64 {
	ttlStr := os.Getenv("IMPOSTER_STORE_DYNAMODB_TTL")
	if ttlStr == "" {
		return -1
	}
	ttl, err := strconv.ParseInt(ttlStr, 10, 64)
	if err != nil {
		logger.Errorf("invalid TTL value: %v", err)
		return -1
	}
	return ttl
}

func getTTLAttributeName() string {
	attributeName := os.Getenv("IMPOSTER_STORE_DYNAMODB_TTL_ATTRIBUTE")
	if attributeName == "" {
		return "ttl"
	}
	return attributeName
}
