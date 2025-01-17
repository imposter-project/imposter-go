package openapi

import "github.com/imposter-project/imposter-go/internal/logger"

// generateExampleJSON generates an example JSON response based on the response object
func generateExampleJSON(response Response) (string, error) {
	if response.Example != "" {
		logger.Debugf("returning example from OpenAPI spec")
		return response.Example, nil
	} else if response.Schema != nil {
		logger.Debugf("generating example from OpenAPI schema")
		// TODO: generate example based on schema
		return "{}", nil
	}
	logger.Warnf("no example or schema found for response")
	return "", nil
}
