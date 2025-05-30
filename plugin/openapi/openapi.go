package openapi

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"net/http"
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

const defaultExampleName = "default-example"

type SparseResponse struct {
	Examples map[string]string
	Schema   *base.SchemaProxy
}

type Response struct {
	UniqueID string
	SparseResponse
	ContentType string
	Headers     map[string]SparseResponse
}

type Operation struct {
	Name      string
	Method    string
	Path      string
	Responses map[int][]Response
}

type parserOptions struct {
	stripServerPath bool
}

type OpenAPIParser interface {
	GetVersion() OpenAPIVersion
	GetOperations() []Operation
	GetOperation(opName string) *Operation
	ValidateRequest(req *http.Request) (bool, []*errors.ValidationError)
	ValidateResponse(rs *exchange.ResponseState) (bool, []*errors.ValidationError)
}

// GetResponse returns a response by its unique ID
func (o Operation) GetResponse(responseId string) *Response {
	var openApiResp *Response
	for _, resp := range o.Responses {
		for _, statusResp := range resp {
			if statusResp.UniqueID == responseId {
				openApiResp = &statusResp
				break
			}
		}
	}
	return openApiResp
}

func newOpenAPIParser(specFile string, validate bool, opts parserOptions) (OpenAPIParser, error) {
	logger.Tracef("loading OpenAPI spec %s", specFile)

	spec, _ := os.ReadFile(specFile)
	document, err := libopenapi.NewDocument(spec)
	if err != nil {
		return nil, fmt.Errorf("cannot create new document: %e", err)
	}

	var oasValidator *validator.Validator
	if validate {
		highLevelValidator, validatorErrs := validator.NewValidator(document)
		if validatorErrs != nil {
			var errorMessages string
			for i := range validatorErrs {
				errorMessages += fmt.Sprintf("error: %e\n", validatorErrs[i])
			}
			return nil, fmt.Errorf("cannot create validator: %d errors reported: %v", len(validatorErrs), errorMessages)
		}
		oasValidator = &highLevelValidator
	}

	if strings.HasPrefix(document.GetSpecInfo().Version, "3") {
		return newOpenAPI3Parser(document, oasValidator, opts)
	} else {
		logger.Tracef("assuming document version is Swagger/OpenAPI 2")
		return newOpenAPI2Parser(document, oasValidator, opts)
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
			// Create an interceptor with default RequestMatcher
			newInterceptor := config.Interceptor{
				Continue: true,
				BaseResource: config.BaseResource{
					RuntimeGenerated: true,
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
						// TODO check request headers, query params, etc.
					},
					Capture: map[string]config.Capture{
						"_matched-openapi-operation": {
							Store: "request",
							CaptureConfig: config.CaptureConfig{
								Const: op.Name,
							},
						},
						"_matched-openapi-response": {
							Store: "request",
							CaptureConfig: config.CaptureConfig{
								Const: resp.UniqueID,
							},
						},
					},
					Response: &config.Response{
						StatusCode: responseCode,
						Headers: map[string]string{
							"Content-Type": resp.ContentType,
						},
						Content: openapiExamplePlaceholder,
					},
				},
			}
			logger.Tracef("adding interceptor for operation %s at %s %s", op.Name, op.Method, op.Path)
			cfg.Interceptors = append(cfg.Interceptors, newInterceptor)
		}
	}

	// Add a default resource to handle unmatched requests
	defaultResource := config.Resource{
		BaseResource: config.BaseResource{
			RuntimeGenerated: true,
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
			Response: &config.Response{},
		},
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
