package openapi

import "github.com/imposter-project/imposter-go/internal/config"

// OpenAPIVersion represents the version of OpenAPI being used
type OpenAPIVersion int

const (
	OpenAPI20 OpenAPIVersion = iota + 1
	OpenAPI30
	OpenAPI31
)

type Operation struct {
	Name                string
	Method              string
	Path                string
	ResponseContentType string
	ResponseSchema      interface{}
}

type OpenAPIParser interface {
	GetVersion() OpenAPIVersion
	GetOperations() []Operation
}

func newOpenAPIParser(specFile string) (*OpenAPIParser, error) {
	return nil, nil
}

// augmentConfigWithOpenApiSpec enriches the configuration with auto-generated interceptors for each OpenAPI operation.
func augmentConfigWithOpenApiSpec(cfg *config.Config, parser OpenAPIParser) error {
	ops := parser.GetOperations()
	for _, op := range ops {
		// Generate example response JSON
		// TODO make this lazy; use a template placeholder function, such as ${soap.example('${op.Name}')}
		exampleResponse, err := generateExampleJSON(op.ResponseSchema, &parser)
		if err != nil {
			return err
		}

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
				// TODO add headers, query params, etc.
			},
			Response: &config.Response{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": op.ResponseContentType,
				},
				Content: exampleResponse,
			},
		}
		cfg.Interceptors = append(cfg.Interceptors, newInterceptor)
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
