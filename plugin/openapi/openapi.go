package openapi

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"os"
	"sort"
	"strings"
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
	Example     string
	Schema      *base.SchemaProxy
}

type Operation struct {
	Name      string
	Method    string
	Path      string
	Responses map[int][]Response
}

type OpenAPIParser interface {
	GetVersion() OpenAPIVersion
	GetOperations() []Operation
}

func newOpenAPIParser(specFile string) (OpenAPIParser, error) {
	spec, _ := os.ReadFile(specFile)
	document, err := libopenapi.NewDocument(spec)
	if err != nil {
		return nil, fmt.Errorf("cannot create new document: %e", err)
	}

	if strings.HasPrefix(document.GetSpecInfo().Version, "3") {
		return newOpenAPI3Parser(document)
	} else {
		logger.Tracef("assuming document version is Swagger/OpenAPI 2")
		return newOpenAPI2Parser(document)
	}
}

// augmentConfigWithOpenApiSpec enriches the configuration with auto-generated interceptors for each OpenAPI operation.
func augmentConfigWithOpenApiSpec(cfg *config.Config, parser OpenAPIParser) error {
	ops := parser.GetOperations()
	for _, op := range ops {
		logger.Debugf("adding interceptor for operation %s %s", op.Method, op.Path)

		responseCode := getDefaultResponseCode(op)
		responses := op.Responses[responseCode]

		for _, resp := range responses {

			// Generate example response JSON
			// TODO make this lazy; use a template placeholder function, such as ${soap.example('${op.Name}')}
			exampleResponse, err := generateExampleJSON(resp)
			if err != nil {
				return err
			}

			// Create an interceptor with default RequestMatcher
			newInterceptor := config.Interceptor{
				Continue: true,
				RequestMatcher: config.RequestMatcher{
					Method: op.Method,
					Path:   op.Path,
					RequestHeaders: map[string]config.MatcherUnmarshaler{
						"Accept": {
							Matcher: config.MatchCondition{
								Value:    resp.ContentType,
								Operator: "Contains",
							},
						},
					},
					Capture: map[string]config.Capture{
						"_matched-openapi-operation": {
							Store: "request",
							CaptureConfig: config.CaptureConfig{
								Const: op.Name,
							},
						},
					},
					// TODO add request headers, query params, etc.
				},
				Response: &config.Response{
					StatusCode: responseCode,
					Headers: map[string]string{
						"Content-Type": resp.ContentType,
					},
					Content: exampleResponse,
					// TODO add response headers
				},
			}
			logger.Tracef("adding interceptor for operation %s at %s %s", op.Name, op.Method, op.Path)
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
						Operator: "Exists",
					},
				},
			},
		},
		Response: config.Response{},
	}
	cfg.Resources = append(cfg.Resources, defaultResource)

	return nil
}

// getDefaultResponseCode guesses the default response code for an operation
func getDefaultResponseCode(op Operation) int {
	var codes []int
	for code := range op.Responses {
		codes = append(codes, code)
	}
	sort.Ints(codes)

	var responseCode int
	for _, code := range codes {
		if code == 200 {
			responseCode = code
			break
		} else if code > 200 {
			responseCode = code
			break
		}
	}
	if responseCode == 0 {
		responseCode = codes[0]
	}
	return responseCode
}
