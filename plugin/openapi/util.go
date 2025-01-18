package openapi

import (
	"github.com/imposter-project/imposter-go/internal/logger"
	"gopkg.in/yaml.v3"
)

// yamlNodeToString converts a YAML node to a string
func yamlNodeToString(node *yaml.Node) string {
	if node != nil {
		if len(node.Content) == 0 {
			return node.Value
		}
		// attempt to marshal the node
		marshalled, err := yaml.Marshal(node)
		if err != nil {
			logger.Warnf("failed to marshal example: %e", err)
		}
		return string(marshalled)
	}
	return ""
}
