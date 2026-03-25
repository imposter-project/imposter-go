package openapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/imposter-project/imposter-go/internal/fakedata"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

const (
	openapiExamplePlaceholder = "${openapi.example()}"
)

// generateExampleRaw generates an example object based on the sparse response object.
// If the response has an example, it will be returned as is, and isLiteral will be true.
// If the response has a schema, an example will be generated based on the schema.
func generateExampleRaw(response SparseResponse, exampleName string) (example interface{}, isLiteral bool, err error) {
	// TODO cache example responses

	var exampleStr string
	if exampleName != "" {
		ex, found := response.Examples[exampleName]
		if !found {
			logger.Warnf("example '%s' not found in spec", exampleName)
		}
		exampleStr = ex
	}
	if exampleStr != "" {
		logger.Debugf("returning example '%s' from OpenAPI spec", exampleName)
		return exampleStr, true, nil
	} else if response.Schema != nil {
		logger.Debugf("generating example from OpenAPI schema")
		example, err = generateExample(response.Schema)
		if err != nil {
			return "", false, fmt.Errorf("failed to generate example: %w", err)
		}
		return example, false, nil
	}
	logger.Warnf("no example or schema found for response")
	return "", false, nil
}

// generateExampleJSON generates an example JSON response based on the sparse response object
func generateExampleJSON(response SparseResponse, exampleName string) (string, error) {
	example, isLiteral, err := generateExampleRaw(response, exampleName)
	if err != nil {
		return "", err
	}
	if isLiteral {
		return example.(string), nil
	}
	exampleJSON, err := json.Marshal(example)
	if err != nil {
		return "", fmt.Errorf("failed to marshal example: %w", err)
	}
	return string(exampleJSON), nil
}

// generateExampleString generates an example string response based on the sparse response object
func generateExampleString(response SparseResponse, exampleName string) (string, error) {
	example, isLiteral, err := generateExampleRaw(response, exampleName)
	if err != nil {
		return "", err
	}
	if isLiteral {
		return example.(string), nil
	}
	return coerceToString(example)
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
	schemaType := getSchemaType(schema)

	// If schema has an example, use it
	if schema.Example != nil {
		schemaExample := yamlNodeToObj(schema.Example)
		return schemaExample, nil
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

	// if schemaType is empty, try to infer it from other schema properties
	if schemaType == "" && schema.Properties != nil {
		schemaType = "object"
	} else if schemaType == "" && schema.Items != nil {
		schemaType = "array"
	}

	// Check for x-fake-data extension before falling through to default generation
	if ext := getFakeDataExtension(schema); ext != "" {
		if val, ok := generateFromFakeDataExtension(ext, schemaType); ok {
			return val, nil
		}
	}

	switch schemaType {
	case "string":
		return generateStringExample(schema)
	case "integer", "number":
		return generateNumberExample(schema)
	case "boolean":
		return generateBooleanExample()
	case "null":
		return nil, nil
	case "array":
		return generateArrayExample(schema)
	case "object":
		return generateObjectExample(schema)
	default:
		return nil, fmt.Errorf("unsupported schema type: %v", schema.Type)
	}
}

func renderExampleAsType(exampleValue string, schemaType string) interface{} {
	switch schemaType {
	case "integer":
		n, _ := strconv.Atoi(exampleValue)
		return n
	case "number":
		n, _ := strconv.ParseFloat(exampleValue, 32)
		return n
	case "boolean":
		return exampleValue == "true"
	case "null":
		return nil
	default:
		return exampleValue
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
		case "byte":
			// base64 encoded string
			return "SW1wb3N0ZXI0bGlmZQo=", nil
		case "date-time":
			if val, ok := fakedata.GenerateForFormat("date-time"); ok {
				return val, nil
			}
			return time.Now().UTC().Format(time.RFC3339), nil
		case "date":
			if val, ok := fakedata.GenerateForFormat("date"); ok {
				return val, nil
			}
			return time.Now().UTC().Format("2006-01-02"), nil
		case "email":
			if val, ok := fakedata.GenerateForFormat("email"); ok {
				return val, nil
			}
			return "test@example.com", nil
		case "password":
			if val, ok := fakedata.GenerateForFormat("password"); ok {
				return val, nil
			}
			return "changeme", nil
		case "uuid":
			return "123e4567-e89b-12d3-a456-426614174000", nil
		case "uri", "url":
			if val, ok := fakedata.GenerateForFormat("uri"); ok {
				return val, nil
			}
			return "https://example.com", nil
		case "hostname":
			if val, ok := fakedata.GenerateForFormat("hostname"); ok {
				return val, nil
			}
			return "example.com", nil
		case "ipv4":
			if val, ok := fakedata.GenerateForFormat("ipv4"); ok {
				return val, nil
			}
			return "192.168.1.1", nil
		case "ipv6":
			if val, ok := fakedata.GenerateForFormat("ipv6"); ok {
				return val, nil
			}
			return "::1", nil
		}
	}
	return "example", nil
}

func generateNumberExample(schema *base.Schema) (interface{}, error) {
	// TODO consider min/max
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
	return false, nil
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
			propSchema := prop.Schema()

			// If property schema has an explicit example, use standard generation
			if propSchema.Example != nil {
				value, err := generateExample(prop)
				if err != nil {
					return nil, fmt.Errorf("failed to generate example for property %s: %w", name, err)
				}
				result[name] = value
				continue
			}

			// Check for x-fake-data extension on the property
			if ext := getFakeDataExtension(propSchema); ext != "" {
				schemaType := getSchemaType(propSchema)
				if val, ok := generateFromFakeDataExtension(ext, schemaType); ok {
					result[name] = val
					continue
				}
			}

			// Try to infer fake data from the property name
			if val, ok := fakedata.GenerateForPropertyName(name); ok {
				schemaType := getSchemaType(propSchema)
				result[name] = renderExampleAsType(val, schemaType)
				continue
			}

			// Fall back to standard example generation
			value, err := generateExample(prop)
			if err != nil {
				return nil, fmt.Errorf("failed to generate example for property %s: %w", name, err)
			}
			result[name] = value
		}
	}

	return result, nil
}

// getFakeDataExtension extracts the x-fake-data extension value from a schema.
// Returns the value (e.g. "Color.name") or empty string if not present.
func getFakeDataExtension(schema *base.Schema) string {
	if schema.Extensions == nil {
		return ""
	}
	node, ok := schema.Extensions.Get("x-fake-data")
	if !ok || node == nil {
		return ""
	}
	return node.Value
}

// generateFromFakeDataExtension generates a fake data value from an x-fake-data
// extension value like "Color.name" or "Name.firstName".
// Returns the generated value rendered as the appropriate type and true if successful.
func generateFromFakeDataExtension(ext string, schemaType string) (interface{}, bool) {
	parts := strings.SplitN(ext, ".", 2)
	if len(parts) != 2 {
		logger.Warnf("invalid x-fake-data value: %s (expected Category.property)", ext)
		return nil, false
	}
	val := fakedata.Generate(parts[0], parts[1])
	if val == "" {
		return nil, false
	}
	return renderExampleAsType(val, schemaType), true
}
