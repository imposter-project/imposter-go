package openapi

import (
	"fmt"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	uuid "github.com/satori/go.uuid"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type openAPI3Parser struct {
	openAPIParser
	version    OpenAPIVersion
	operations []Operation
}

// newOpenAPI3Parser creates a new OpenAPIParser for OpenAPI 3 documents
func newOpenAPI3Parser(document libopenapi.Document, validator *validator.Validator, opts parserOptions) (*openAPI3Parser, error) {
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
		openAPIParser: openAPIParser{
			validator: validator,
		},
		version: version,
	}
	if err := parser.parseOperations(v3Model, opts); err != nil {
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

func (p *openAPI3Parser) GetOperation(opName string) *Operation {
	for _, op := range p.operations {
		if op.Name == opName {
			return &op
		}
	}
	return nil
}

// parseOperations extracts operations from the OpenAPI 3 document
func (p *openAPI3Parser) parseOperations(v3Model *libopenapi.DocumentModel[v3.Document], opts parserOptions) error {
	paths := v3Model.Model.Paths.PathItems.Len()
	var schemas int
	if v3Model.Model.Components != nil && v3Model.Model.Components.Schemas != nil {
		schemas = v3Model.Model.Components.Schemas.Len()
	}
	logger.Debugf("found %d paths and %d schemas in the specification", paths, schemas)

	for path, pathItem := range v3Model.Model.Paths.PathItems.FromOldest() {
		finalPath := p.getFinalPath(v3Model.Model.Servers, opts.stripServerPath, path)

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
			r := SparseResponse{
				Examples: make(map[string]string),
				Schema:   content.Schema,
			}
			if content.Example != nil {
				r.Examples[defaultExampleName] = yamlNodeToJson(content.Example)
			} else if content.Examples != nil && content.Examples.Len() > 0 {
				for exampleName, ex := range content.Examples.FromOldest() {
					r.Examples[exampleName] = yamlNodeToJson(ex.Value)
				}
			}

			respHeaders := make(map[string]SparseResponse)
			for headerName, header := range resp.Headers.FromOldest() {
				h := SparseResponse{
					Examples: make(map[string]string),
					Schema:   header.Schema,
				}
				if header.Example != nil {
					h.Examples[defaultExampleName] = yamlNodeToString(header.Example)
				} else if header.Examples != nil && header.Examples.Len() > 0 {
					// we only support one example for headers
					ex := header.Examples.Oldest().Value
					h.Examples[defaultExampleName] = yamlNodeToString(ex.Value)
				}
				respHeaders[headerName] = h
			}

			response := Response{
				UniqueID:       uuid.NewV4().String(),
				ContentType:    mediaType,
				SparseResponse: r,
				Headers:        respHeaders,
			}
			responses = append(responses, response)
		}
	}
	return responses
}

func (p *openAPI3Parser) getFinalPath(servers []*v3.Server, stripServerPath bool, path string) string {
	if stripServerPath {
		return path
	}
	if len(servers) == 0 {
		return path
	}
	server := servers[0]

	// extract the path portion of server.URL
	var serverPath string
	serverUrl, err := url.Parse(server.URL)
	if err != nil {
		logger.Warnf("failed to parse server URL: %s", server.URL)
		serverPath = ""
	} else {
		serverPath = serverUrl.Path
	}
	return fmt.Sprintf("%s%s", serverPath, path)
}
