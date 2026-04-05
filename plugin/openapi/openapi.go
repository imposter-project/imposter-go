package openapi

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/pkg/feature"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
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
	stripServerPath          bool
	externalReferenceBaseURL string
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

	// Build the libopenapi document configuration up front so external
	// $ref resolution honours the feature flags and any configured
	// external reference base URL.
	//
	// libopenapi drives file-ref resolution from BasePath (set to the
	// spec file's directory so refs resolve relative to the spec, not
	// the process cwd) and drives remote-ref resolution from BaseURL.
	// The AllowRemoteReferences escape hatch forces construction of a
	// remote filesystem even when no BaseURL has been supplied - this
	// is what lets users opt into absolute http(s) $refs.
	docCfg := &datamodel.DocumentConfiguration{}
	if feature.Bool(flagAllowFileRefs) {
		docCfg.BasePath = filepath.Dir(specFile)
		docCfg.SpecFilePath = specFile
	} else {
		logger.Infof("OpenAPI file $ref resolution disabled (set %s=true to enable)", flagAllowFileRefs.EnvVar)
	}
	if feature.Bool(flagAllowRemoteRefs) {
		docCfg.AllowRemoteReferences = true
	} else {
		logger.Infof("OpenAPI remote $ref resolution disabled (set %s=true to enable)", flagAllowRemoteRefs.EnvVar)
	}
	if opts.externalReferenceBaseURL != "" {
		u, err := url.Parse(opts.externalReferenceBaseURL)
		if err != nil {
			return nil, fmt.Errorf("cannot parse external reference URL: %w", err)
		}
		docCfg.BaseURL = u
		logger.Infof("external base URL set to: %s", u.String())
	}

	document, err := libopenapi.NewDocumentWithConfiguration(spec, docCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create new document: %w", err)
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
			// Build request matcher with method and path
			reqMatcher := config.RequestMatcher{
				Method: op.Method,
				Path:   op.Path,
				// TODO check request headers, query params, etc.
			}

			// Only add Accept header matching if content type is specified.
			// Use AnyOf so that requests without an Accept header still match,
			// since the client has not expressed a representation constraint.
			if resp.ContentType != "" {
				reqMatcher.AnyOf = []config.ExpressionMatchCondition{
					{
						Expression: "${context.request.headers.Accept}",
						MatchCondition: config.MatchCondition{
							Operator: "Contains",
							Value:    resp.ContentType,
						},
					},
					{
						Expression: "${context.request.headers.Accept}",
						MatchCondition: config.MatchCondition{
							Operator: "EqualTo",
							Value:    "*/*",
						},
					},
					{
						Expression: "${context.request.headers.Accept}",
						MatchCondition: config.MatchCondition{
							Operator: "NotExists",
						},
					},
				}
			}

			// Create an interceptor with default RequestMatcher
			newInterceptor := config.Interceptor{
				Continue: true,
				BaseResource: config.BaseResource{
					RuntimeGenerated: true,
					RequestMatcher:   reqMatcher,
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
