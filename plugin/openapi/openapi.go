package openapi

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"os"
	"strconv"
)

// OpenAPIVersion represents the version of OpenAPI being used
type OpenAPIVersion int

const (
	OpenAPI20 OpenAPIVersion = iota + 1
	OpenAPI30
	OpenAPI31
)

type Response struct {
	ContentType string

	// TODO don't use libopenapi types directly
	MediaType *v3.MediaType
}

type Operation struct {
	Name      string
	Method    string
	Path      string
	Responses map[string]Response
}

type openAPIParser struct {
	version    OpenAPIVersion
	operations []Operation
}

type OpenAPIParser interface {
	GetVersion() OpenAPIVersion
	GetOperations() []Operation
}

func (p openAPIParser) GetVersion() OpenAPIVersion {
	return p.version
}

func (p openAPIParser) GetOperations() []Operation {
	return p.operations
}

func newOpenAPIParser(specFile string) (OpenAPIParser, error) {
	petstore, _ := os.ReadFile(specFile)
	document, err := libopenapi.NewDocument(petstore)
	if err != nil {
		return nil, fmt.Errorf("cannot create new document: %e", err)
	}

	// TODO determine OpenAPI version
	v3Model, errors := document.BuildV3Model()

	if len(errors) > 0 {
		var errorMessages string
		for i := range errors {
			errorMessages += fmt.Sprintf("error: %e\n", errors[i])
		}
		return nil, fmt.Errorf("cannot create v3 model from document: %d errors reported: %v", len(errors), errorMessages)
	}

	// get a count of the number of paths and schemas.
	paths := v3Model.Model.Paths.PathItems.Len()
	schemas := v3Model.Model.Components.Schemas.Len()

	// print the number of paths and schemas in the document
	logger.Debugf("there are %d paths and %d schemas in the document", paths, schemas)

	parser := openAPIParser{}

	for path, pathItem := range v3Model.Model.Paths.PathItems.FromOldest() {
		operations := pathItem.GetOperations()
		for method, operation := range operations.FromOldest() {
			operationName := fmt.Sprintf("%s %s", method, path)
			op := Operation{
				Name:      operationName,
				Path:      path,
				Method:    method,
				Responses: make(map[string]Response),
			}

			for code, resp := range operation.Responses.Codes.FromOldest() {
				for mediaType, content := range resp.Content.FromOldest() {
					op.Responses[code] = Response{
						ContentType: mediaType,
						MediaType:   content,
					}
				}
			}

			parser.operations = append(parser.operations, op)
		}
	}

	return parser, nil
}

// augmentConfigWithOpenApiSpec enriches the configuration with auto-generated interceptors for each OpenAPI operation.
func augmentConfigWithOpenApiSpec(cfg *config.Config, parser OpenAPIParser) error {
	ops := parser.GetOperations()
	for _, op := range ops {
		for code, resp := range op.Responses {
			// Generate example response JSON
			// TODO make this lazy; use a template placeholder function, such as ${soap.example('${op.Name}')}
			exampleResponse, err := generateExampleJSON(op.Responses, &parser)
			if err != nil {
				return err
			}

			statusCode, _ := strconv.Atoi(code)

			// Create an interceptor with default RequestMatcher
			newInterceptor := config.Interceptor{
				Continue: true,
				RequestMatcher: config.RequestMatcher{
					Method: op.Method,
					Path:   op.Path,
					Capture: map[string]config.Capture{
						"_matched-openapi-operation": {
							Store: "request",
							CaptureConfig: config.CaptureConfig{
								Const: "true",
							},
						},
					},
					RequestHeaders: map[string]config.MatcherUnmarshaler{
						"Accept": {
							Matcher: config.MatchCondition{
								Value:    resp.ContentType,
								Operator: "Contains",
							},
						},
					},
					// TODO add request headers, query params, etc.
				},
				Response: &config.Response{
					StatusCode: statusCode,
					Headers: map[string]string{
						"Content-Type": resp.ContentType,
					},
					Content: exampleResponse,
					// TODO add response headers
				},
			}
			cfg.Interceptors = append(cfg.Interceptors, newInterceptor)
		}
	}

	// Add a default resource to handle unmatched requests
	defaultResource := config.Resource{
		RequestMatcher: config.RequestMatcher{
			AllOf: []config.ExpressionMatchCondition{
				{
					Expression: "${stores.request._matched-openapi-operation}",
					MatchCondition: config.MatchCondition{
						Operator: "EqualTo",
						Value:    "true",
					},
				},
			},
		},
		Response: config.Response{},
	}
	cfg.Resources = append(cfg.Resources, defaultResource)

	return nil
}
