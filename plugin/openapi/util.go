package openapi

import (
	"encoding/json"
	"github.com/imposter-project/imposter-go/internal/logger"
	"gopkg.in/yaml.v3"
)

// yamlNodeToJson converts a YAML node to a string
func yamlNodeToJson(node *yaml.Node) string {
	if node != nil {
		ex := yamlNodeToObj(node)
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
		if len(node.Content) == 0 {
			return node.Value
		}
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
