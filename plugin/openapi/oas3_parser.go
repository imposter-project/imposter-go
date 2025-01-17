package openapi

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"net/http"
	"strconv"
	"strings"
)

type openAPI3Parser struct {
	version    OpenAPIVersion
	operations []Operation
}

// newOpenAPI3Parser creates a new OpenAPIParser for OpenAPI 3 documents
func newOpenAPI3Parser(document libopenapi.Document) (*openAPI3Parser, error) {
	logger.Debugf("creating OpenAPI 3 parser")
	v3Model, errors := document.BuildV3Model()

	if len(errors) > 0 {
		var errorMessages string
		for i := range errors {
			errorMessages += fmt.Sprintf("error: %e\n", errors[i])
		}
		return nil, fmt.Errorf("cannot create v3 model from document: %d errors reported: %v", len(errors), errorMessages)
	}

	var version OpenAPIVersion
	if strings.HasPrefix(document.GetSpecInfo().Version, "3.1") {
		version = OpenAPI31
	} else {
		version = OpenAPI30
	}
	parser := &openAPI3Parser{
		version: version,
	}
	if err := parser.parseOperations(v3Model); err != nil {
		return nil, fmt.Errorf("cannot parse operations: %e", err)
	}
	return parser, nil
}

func (p *openAPI3Parser) GetVersion() OpenAPIVersion {
	return p.version
}

func (p *openAPI3Parser) GetOperations() []Operation {
	return p.operations
}

// parseOperations extracts operations from the OpenAPI 3 document
func (p *openAPI3Parser) parseOperations(v3Model *libopenapi.DocumentModel[v3.Document]) error {
	paths := v3Model.Model.Paths.PathItems.Len()
	var schemas int
	if v3Model.Model.Components != nil && v3Model.Model.Components.Schemas != nil {
		schemas = v3Model.Model.Components.Schemas.Len()
	}
	logger.Debugf("found %d paths and %d schemas in the specification", paths, schemas)

	for path, pathItem := range v3Model.Model.Paths.PathItems.FromOldest() {
		operations := pathItem.GetOperations()
		for method, operation := range operations.FromOldest() {
			operationName := fmt.Sprintf("%s %s", method, path)
			op := Operation{
				Name:      operationName,
				Path:      path,
				Method:    method,
				Responses: make(map[int][]Response),
			}

			if operation.Responses.Default != nil {
				// note: this might be overwritten by a more specific 200 response
				op.Responses[http.StatusOK] = p.processResponse(operation.Responses.Default)
			}

			for code, resp := range operation.Responses.Codes.FromOldest() {
				statusCode, _ := strconv.Atoi(code)
				op.Responses[statusCode] = p.processResponse(resp)
			}

			p.operations = append(p.operations, op)
		}
	}
	return nil
}

// processResponse converts an OpenAPI 3 response into a list of Response objects
func (p *openAPI3Parser) processResponse(resp *v3.Response) []Response {
	responses := make([]Response, 0)
	if resp.Content == nil || resp.Content.Len() == 0 {
		responses = []Response{
			{
				ContentType: "",
			},
		}
	} else {
		for mediaType, content := range resp.Content.FromOldest() {
			var example string
			if content.Example != nil {
				example = content.Example.Value
			} else if content.Examples != nil && content.Examples.Len() > 0 {
				ex := content.Examples.Oldest().Value
				example = ex.ExternalValue
			}

			response := Response{
				ContentType: mediaType,
				Example:     example,
				Schema:      content.Schema,
			}
			responses = append(responses, response)
		}
	}
	return responses
}
