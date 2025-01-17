package openapi

import "github.com/imposter-project/imposter-go/internal/logger"

func generateExampleJSON(response Response, parser *OpenAPIParser) (string, error) {
	if response.MediaType == nil {
		return "", nil
	}
	if response.MediaType.Example != nil {
		logger.Debugf("returning example from OpenAPI spec")
		return response.MediaType.Example.Value, nil
	} else if response.MediaType.Examples != nil && response.MediaType.Examples.Len() > 0 {
		logger.Debugf("returning example from OpenAPI spec")
		example := response.MediaType.Examples.Oldest().Value
		return example.ExternalValue, nil
	} else {
		// TODO: generate example based on schema
	}
	return "", nil
}
