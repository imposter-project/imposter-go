package openapi

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// generateExampleJSON generates an example JSON response based on the response object
func generateExampleJSON(response Response) (string, error) {
	if response.Example != "" {
		logger.Debugf("returning example from OpenAPI spec")
		return response.Example, nil
	} else if response.Schema != nil {
		logger.Debugf("generating example from OpenAPI schema")
		example, err := generateExample(response.Schema)
		if err != nil {
			return "", fmt.Errorf("failed to generate example: %w", err)
		}
		exampleJSON, err := json.Marshal(example)
		if err != nil {
			return "", fmt.Errorf("failed to marshal example: %w", err)
		}
		return string(exampleJSON), nil
	}
	logger.Warnf("no example or schema found for response")
	return "", nil
}

// getSchemaType returns the first type from the schema's Type array
func getSchemaType(schema *base.Schema) string {
	if len(schema.Type) > 0 {
		return schema.Type[0]
	}
	return ""
}

// generateExample generates an example value based on the schema
func generateExample(schemaProxy *base.SchemaProxy) (interface{}, error) {
	schema := schemaProxy.Schema()

	// If schema has an example, use it
	if schema.Example != nil {
		return schema.Example.Value, nil
	}

	// If schema has an enum, use the first value
	if len(schema.Enum) > 0 {
		enumNode := schema.Enum[0]
		return enumNode.Value, nil
	}

	// Handle composition keywords
	if schema.AllOf != nil && len(schema.AllOf) > 0 {
		return generateAllOfExample(schema.AllOf)
	}
	if schema.OneOf != nil && len(schema.OneOf) > 0 {
		return generateExample(schema.OneOf[0]) // Pick first schema
	}
	if schema.AnyOf != nil && len(schema.AnyOf) > 0 {
		return generateExample(schema.AnyOf[0]) // Pick first schema
	}

	// Handle different types
	switch getSchemaType(schema) {
	case "string":
		return generateStringExample(schema)
	case "integer", "number":
		return generateNumberExample(schema)
	case "boolean":
		return generateBooleanExample()
	case "array":
		return generateArrayExample(schema)
	case "object":
		return generateObjectExample(schema)
	default:
		return nil, fmt.Errorf("unsupported schema type: %v", schema.Type)
	}
}

// generateAllOfExample merges examples from all schemas in allOf
func generateAllOfExample(schemas []*base.SchemaProxy) (interface{}, error) {
	result := make(map[string]interface{})

	for _, schema := range schemas {
		example, err := generateExample(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to generate allOf example: %w", err)
		}

		// If example is a map, merge it with result
		if exampleMap, ok := example.(map[string]interface{}); ok {
			for k, v := range exampleMap {
				result[k] = v
			}
		}
	}

	return result, nil
}

func generateStringExample(schema *base.Schema) (string, error) {
	if schema.Format != "" {
		switch schema.Format {
		case "date-time":
			return time.Now().UTC().Format(time.RFC3339), nil
		case "date":
			return time.Now().UTC().Format("2006-01-02"), nil
		case "email":
			return "user@example.com", nil
		case "uuid":
			return "123e4567-e89b-12d3-a456-426614174000", nil
		}
	}
	return "example", nil
}

func generateNumberExample(schema *base.Schema) (interface{}, error) {
	if schema.Format == "int64" {
		return int64(42), nil
	}
	if schema.Format == "int32" {
		return int32(42), nil
	}
	if schema.Format == "float" || schema.Format == "double" {
		return 42.42, nil
	}
	return 42, nil
}

func generateBooleanExample() (bool, error) {
	return true, nil
}

func generateArrayExample(schema *base.Schema) ([]interface{}, error) {
	if schema.Items == nil {
		return nil, fmt.Errorf("array schema missing items")
	}

	// Get the schema from DynamicValue
	itemSchema := schema.Items.A

	// Generate one example item
	item, err := generateExample(itemSchema)
	if err != nil {
		return nil, err
	}

	return []interface{}{item}, nil
}

func generateObjectExample(schema *base.Schema) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Handle properties from OrderedMap
	if schema.Properties != nil {
		for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
			name := pair.Key()
			prop := pair.Value()
			value, err := generateExample(prop)
			if err != nil {
				return nil, fmt.Errorf("failed to generate example for property %s: %w", name, err)
			}
			result[name] = value
		}
	}

	return result, nil
}
