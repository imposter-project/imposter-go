package query

import (
	"encoding/json"
	"strings"

	"github.com/PaesslerAG/jsonpath"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// isUnknownKeyError checks if the error is related to an unknown key in the JSON document.
func isUnknownKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), "unknown key")
}

// JsonPathQuery extracts a value from the JSON document using a JSONPath expression.
func JsonPathQuery(doc []byte, jsonPathExpr string) (result interface{}, success bool) {
	var jsonData interface{}
	if err := json.Unmarshal(doc, &jsonData); err != nil {
		logger.Warnf("failed to unmarshal JSON data: %v", err)
		return nil, false
	}
	result, err := jsonpath.Get(jsonPathExpr, jsonData)
	if err != nil {
		// Check if the error is about an unknown key
		if isUnknownKeyError(err) {
			// Return nil with success=true for non-existent keys
			return nil, true
		}
		logger.Warnf("failed to extract JSON path: %v", err)
		return nil, false
	}
	return result, true
}
