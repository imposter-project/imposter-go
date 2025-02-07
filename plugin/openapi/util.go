package openapi

import (
	"encoding/json"
	"fmt"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"gopkg.in/yaml.v3"
)

// yamlNodeToJson converts a YAML node to a string
func yamlNodeToJson(node *yaml.Node) string {
	if node != nil {
		ex := yamlNodeToObj(node)
		if ex == nil {
			return ""
		}
		jsonEx, err := json.Marshal(ex)
		if err != nil {
			logger.Warnf("failed to marshal example: %e", err)
			return ""
		}
		return string(jsonEx)
	}
	return ""
}

// yamlNodeToObj converts a YAML node to a go object
func yamlNodeToObj(node *yaml.Node) interface{} {
	if node != nil {
		var temp interface{}
		err := node.Decode(&temp)
		if err != nil {
			logger.Warnf("failed to decode example: %e", err)
			return nil
		}
		return temp
	}
	return nil
}

// yamlNodeToString converts a YAML node to a string
func yamlNodeToString(node *yaml.Node) string {
	obj := yamlNodeToObj(node)
	str, err := coerceToString(obj)
	if err != nil {
		logger.Warnf("failed to convert %v to string: %e", obj, err)
		return ""
	}
	return str
}

// coerceToString converts a simple scalar type to a string
func coerceToString(in interface{}) (string, error) {
	switch in.(type) {
	case string:
		return in.(string), nil
	case int, int32, int64:
		return fmt.Sprintf("%d", in), nil
	case float64, float32:
		return fmt.Sprintf("%f", in), nil
	case bool:
		return fmt.Sprintf("%t", in), nil
	default:
		return "", fmt.Errorf("unsupported example type: %T", in)
	}
}
