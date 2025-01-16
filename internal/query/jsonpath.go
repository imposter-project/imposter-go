package query

import (
	"encoding/json"
	"github.com/PaesslerAG/jsonpath"
	"github.com/imposter-project/imposter-go/internal/logger"
)

// JsonPathQuery extracts a value from the JSON document using a JSONPath expression.
func JsonPathQuery(doc []byte, jsonPathExpr string) (result interface{}, success bool) {
	var jsonData interface{}
	if err := json.Unmarshal(doc, &jsonData); err != nil {
		logger.Warnf("failed to unmarshal JSON data: %v", err)
		return nil, false
	}
	result, err := jsonpath.Get(jsonPathExpr, jsonData)
	if err != nil {
		logger.Warnf("failed to extract JSON path: %v", err)
		return nil, false
	}
	return result, true
}
