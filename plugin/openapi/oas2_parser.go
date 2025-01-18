package openapi

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/pb33f/libopenapi"
	v2 "github.com/pb33f/libopenapi/datamodel/high/v2"
	"net/http"
	"strconv"
	"strings"
)

type openAPI2Parser struct {
	operations []Operation
}

// newOpenAPI2Parser creates a new OpenAPIParser for OpenAPI 2 documents
func newOpenAPI2Parser(document libopenapi.Document, opts parserOptions) (*openAPI2Parser, error) {
	logger.Debugf("creating OpenAPI 2 parser")
	v2Model, errors := document.BuildV2Model()

	if len(errors) > 0 {
		var errorMessages string
		for i := range errors {
			errorMessages += fmt.Sprintf("error: %e\n", errors[i])
		}
		return nil, fmt.Errorf("cannot create v2 model from document: %d errors reported: %v", len(errors), errorMessages)
	}

	parser := &openAPI2Parser{}
	if err := parser.parseOperations(v2Model, opts); err != nil {
		return nil, fmt.Errorf("cannot parse operations: %e", err)
	}
	return parser, nil
}

func (p *openAPI2Parser) GetVersion() OpenAPIVersion {
	return OpenAPI20
}

func (p *openAPI2Parser) GetOperations() []Operation {
	return p.operations
}

// parseOperations extracts operations from the OpenAPI 2 document
func (p *openAPI2Parser) parseOperations(v2Model *libopenapi.DocumentModel[v2.Swagger], opts parserOptions) error {
	paths := v2Model.Model.Paths.PathItems.Len()
	var definitions int
	if v2Model.Model.Definitions != nil {
		definitions = v2Model.Model.Definitions.Definitions.Len()
	}
	logger.Debugf("found %d paths and %d definitions in the specification", paths, definitions)

	for path, pathItem := range v2Model.Model.Paths.PathItems.FromOldest() {
		finalPath := p.getFinalPath(v2Model.Model.BasePath, opts.stripServerPath, path)

		operations := pathItem.GetOperations()
		for method, operation := range operations.FromOldest() {
			operationName := fmt.Sprintf("%s %s", method, finalPath)
			op := Operation{
				Name:      operationName,
				Path:      finalPath,
				Method:    method,
				Responses: make(map[int][]Response),
			}

			if operation.Responses.Default != nil {
				// note: this might be overwritten by a more specific 200 response
				op.Responses[http.StatusOK] = p.processResponse(operation.Produces, operation.Responses.Default)
			}

			for code, resp := range operation.Responses.Codes.FromOldest() {
				statusCode, _ := strconv.Atoi(code)
				op.Responses[statusCode] = p.processResponse(operation.Produces, resp)
			}

			p.operations = append(p.operations, op)
		}
	}
	return nil
}

// processResponse converts an OpenAPI 2 response into a list of Response objects
func (p *openAPI2Parser) processResponse(produces []string, resp *v2.Response) []Response {
	responses := make([]Response, 0)

	// check the produces list first, but it's not mandatory
	for _, mediaType := range produces {
		var example string
		if resp.Examples != nil {
			// get the example for the specific media type, or the first one
			ex, present := resp.Examples.Values.Get(mediaType)
			if !present {
				ex = resp.Examples.Values.Oldest().Value
			}
			example = ex.Value
		}

		response := Response{
			ContentType: mediaType,
			Example:     example,
			Schema:      resp.Schema,
		}
		responses = append(responses, response)
	}

	// derive from response example names
	if resp.Examples != nil {
		for exampleName, ex := range resp.Examples.Values.FromOldest() {
			if !strings.Contains(exampleName, "/") {
				// skip example names that are not media types
				continue
			}
			if doesResponseWithContentTypeExist(responses, exampleName) {
				// skip examples that are already in the produces list
				continue
			}
			example := yamlNodeToString(ex)
			response := Response{
				ContentType: exampleName,
				Example:     example,
				Schema:      resp.Schema,
			}
			responses = append(responses, response)
		}
	}
	if len(responses) == 0 {
		responses = []Response{
			{
				ContentType: "",
				Schema:      resp.Schema,
			},
		}
	}
	return responses
}

// doesResponseWithContentTypeExist checks if a response with the given content type already exists in the list
func doesResponseWithContentTypeExist(responses []Response, contentType string) bool {
	for _, response := range responses {
		if response.ContentType == contentType {
			return true
		}
	}
	return false
}

func (p *openAPI2Parser) getFinalPath(specBasePath string, stripServerPath bool, path string) string {
	if stripServerPath {
		return path
	}
	return fmt.Sprintf("%s%s", specBasePath, path)
}
