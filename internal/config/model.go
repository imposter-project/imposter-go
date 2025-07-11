package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/dlclark/regexp2"
)

// Response represents an HTTP response
type Response struct {
	Content    string            `yaml:"content"`
	StatusCode int               `yaml:"statusCode"`
	Dir        string            `yaml:"dir"`
	File       string            `yaml:"file"`
	Fail       string            `yaml:"fail"`
	Delay      Delay             `yaml:"delay"`
	Headers    map[string]string `yaml:"headers"`
	Template   bool              `yaml:"template"`

	// SOAP-specific fields
	SoapFault bool `yaml:"soapFault"`

	// OpenAPI-specific fields
	ExampleName string `yaml:"exampleName"`
}

// Delay represents the delay configuration for a response
type Delay struct {
	Exact int `yaml:"exact"`
	Min   int `yaml:"min"`
	Max   int `yaml:"max"`
}

// Matcher represents anything that can be matched against a value
type Matcher interface {
	Match(actualValue string) bool
}

// StringMatcher is a simple string matcher that checks for exact equality
type StringMatcher string

func (s StringMatcher) Match(actualValue string) bool {
	return string(s) == actualValue
}

// MatchCondition represents a condition for matching requests
type MatchCondition struct {
	Value    string `yaml:"value"`
	Operator string `yaml:"operator"`
}

func (m MatchCondition) Match(actualValue string) bool {
	switch m.Operator {
	case "EqualTo", "":
		return actualValue == m.Value
	case "NotEqualTo":
		return actualValue != m.Value
	case "Exists":
		return actualValue != ""
	case "NotExists":
		return actualValue == ""
	case "Contains":
		return strings.Contains(actualValue, m.Value)
	case "NotContains":
		return !strings.Contains(actualValue, m.Value)
	case "Matches":
		re := regexp2.MustCompile(m.Value, 0)
		matched, _ := re.MatchString(actualValue)
		return matched
	case "NotMatches":
		re := regexp2.MustCompile(m.Value, 0)
		matched, _ := re.MatchString(actualValue)
		return !matched
	default:
		return false
	}
}

// BodyMatchCondition represents a condition for matching request bodies
type BodyMatchCondition struct {
	MatchCondition `yaml:",inline"`
	JSONPath       string            `yaml:"jsonPath,omitempty"`
	XPath          string            `yaml:"xPath,omitempty"`
	XMLNamespaces  map[string]string `yaml:"xmlNamespaces"`
}

func (b BodyMatchCondition) Match(actualValue string) bool {
	return b.MatchCondition.Match(actualValue)
}

// RequestBody represents the request body matching configuration
type RequestBody struct {
	*BodyMatchCondition `yaml:",inline"`
	AllOf               []BodyMatchCondition `yaml:"allOf"`
	AnyOf               []BodyMatchCondition `yaml:"anyOf"`
}

// ConcurrencyLimit represents a concurrency limit with its associated response
type ConcurrencyLimit struct {
	Threshold int       `yaml:"threshold" json:"threshold"`
	Response  *Response `yaml:"response" json:"response"`
}

// UnmarshalYAML implements custom unmarshaling for RequestBody
func (rb *RequestBody) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type bodyMatchFields struct {
		Value         string            `yaml:"value"`
		Operator      string            `yaml:"operator"`
		XPath         string            `yaml:"xPath"`
		JSONPath      string            `yaml:"jsonPath"`
		XMLNamespaces map[string]string `yaml:"xmlNamespaces"`
	}

	type requestBodyFields struct {
		BodyMatchFields bodyMatchFields      `yaml:",inline"`
		AllOf           []BodyMatchCondition `yaml:"allOf"`
		AnyOf           []BodyMatchCondition `yaml:"anyOf"`
	}

	var fields requestBodyFields

	if err := unmarshal(&fields); err != nil {
		return err
	}

	rb.BodyMatchCondition = &BodyMatchCondition{
		MatchCondition: MatchCondition{
			Value:    fields.BodyMatchFields.Value,
			Operator: fields.BodyMatchFields.Operator,
		},
		JSONPath:      fields.BodyMatchFields.JSONPath,
		XPath:         fields.BodyMatchFields.XPath,
		XMLNamespaces: fields.BodyMatchFields.XMLNamespaces,
	}
	rb.AllOf = fields.AllOf
	rb.AnyOf = fields.AnyOf

	return nil
}

func (rb RequestBody) Match(actualValue string) bool {
	if rb.BodyMatchCondition != nil {
		return rb.BodyMatchCondition.Match(actualValue)
	}
	return false
}

// Capture defines how to capture request data for later use in the response
type Capture struct {
	Enabled       *bool         `yaml:"enabled,omitempty"`
	Store         string        `yaml:"store"`
	Key           CaptureConfig `yaml:"key,omitempty"`
	CaptureConfig `yaml:",inline"`
}

// CaptureConfig represents the key configuration for capturing request data.
type CaptureConfig struct {
	PathParam     string `yaml:"pathParam,omitempty"`
	QueryParam    string `yaml:"queryParam,omitempty"`
	FormParam     string `yaml:"formParam,omitempty"`
	RequestHeader string `yaml:"requestHeader,omitempty"`
	Expression    string `yaml:"expression,omitempty"`
	Const         string `yaml:"const,omitempty"`
	RequestBody   struct {
		JSONPath      string            `yaml:"jsonPath,omitempty"`
		XPath         string            `yaml:"xPath,omitempty"`
		XMLNamespaces map[string]string `yaml:"xmlNamespaces,omitempty"`
	} `yaml:"requestBody,omitempty"`
}

// ExpressionMatchCondition represents a condition for evaluating expressions
type ExpressionMatchCondition struct {
	MatchCondition `yaml:",inline"`
	Expression     string `yaml:"expression"`
}

// RequestMatcher contains the common fields for matching requests
type RequestMatcher struct {
	Method         string                        `yaml:"method"`
	Path           string                        `yaml:"path"`
	QueryParams    map[string]MatcherUnmarshaler `yaml:"queryParams"`
	RequestHeaders map[string]MatcherUnmarshaler `yaml:"requestHeaders"`
	RequestBody    RequestBody                   `yaml:"requestBody"`
	FormParams     map[string]MatcherUnmarshaler `yaml:"formParams"`
	PathParams     map[string]MatcherUnmarshaler `yaml:"pathParams"`
	AllOf          []ExpressionMatchCondition    `yaml:"allOf,omitempty"`
	AnyOf          []ExpressionMatchCondition    `yaml:"anyOf,omitempty"`

	// SOAP-specific fields
	Operation  string `yaml:"operation,omitempty"`
	SOAPAction string `yaml:"soapAction,omitempty"`
	Binding    string `yaml:"binding,omitempty"`
}

// StepType represents the type of step to execute
type StepType string

const (
	ScriptStepType StepType = "script"
	RemoteStepType StepType = "remote"
)

// Step represents a step that can be executed when processing a request
type Step struct {
	Type StepType `yaml:"type"`

	// Script-specific fields
	Lang string `yaml:"lang,omitempty"`
	Code string `yaml:"code,omitempty"`
	File string `yaml:"file,omitempty"`

	// Remote-specific fields
	URL     string             `yaml:"url,omitempty"`
	Method  string             `yaml:"method,omitempty"`
	Headers map[string]string  `yaml:"headers,omitempty"`
	Body    string             `yaml:"body,omitempty"`
	Capture map[string]Capture `yaml:"capture,omitempty"`
}

type BaseResource struct {
	RequestMatcher   `yaml:",inline"`
	Capture          map[string]Capture `yaml:"capture,omitempty"`
	Steps            []Step             `yaml:"steps,omitempty"`
	Response         *Response          `yaml:"response,omitempty"`
	Concurrency      []ConcurrencyLimit `yaml:"concurrency,omitempty"`
	Log              string             `yaml:"log,omitempty"`
	RuntimeGenerated bool               `yaml:"-"`

	// ResourceID is computed at startup
	ResourceID string `yaml:"-"`
}

// Resource represents an HTTP resource
type Resource struct {
	BaseResource `yaml:",inline"`
	Security     *SecurityConfig `yaml:"security,omitempty"`
}

// Interceptor represents an HTTP interceptor that can be executed before resources
type Interceptor struct {
	BaseResource `yaml:",inline"`
	Continue     bool `yaml:"continue"`
}

type System struct {
	Stores        map[string]StoreDefinition `yaml:"stores"`
	XMLNamespaces map[string]string          `yaml:"xmlNamespaces,omitempty"`
}

type StoreDefinition struct {
	PreloadFile string                 `yaml:"preloadFile,omitempty"`
	PreloadData map[string]interface{} `yaml:"preloadData,omitempty"`
}

// CorsConfig represents CORS configuration for the mock server
type CorsConfig struct {
	// AllowOrigins can be a string ("all", "*") or a list of origins
	AllowOrigins interface{} `yaml:"allowOrigins,omitempty"`
	// AllowHeaders is a list of allowed headers
	AllowHeaders []string `yaml:"allowHeaders,omitempty"`
	// AllowMethods is a list of allowed HTTP methods
	AllowMethods []string `yaml:"allowMethods,omitempty"`
	// MaxAge is the number of seconds to cache preflight responses
	MaxAge int `yaml:"maxAge,omitempty"`
	// AllowCredentials indicates whether the request can include user credentials
	AllowCredentials bool `yaml:"allowCredentials,omitempty"`
}

// GetAllowedOrigins returns the allowed origins as a slice of strings
func (c *CorsConfig) GetAllowedOrigins() []string {
	switch v := c.AllowOrigins.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		origins := make([]string, len(v))
		for i, origin := range v {
			if str, ok := origin.(string); ok {
				origins[i] = str
			}
		}
		return origins
	default:
		return nil
	}
}

// ValidationBehaviour defines the behaviour when validation issues occur
type ValidationBehaviour string

const (
	// ValidationBehaviourFail causes requests to fail when validation issues occur
	ValidationBehaviourFail ValidationBehaviour = "fail"
	// ValidationBehaviourLog logs validation issues but allows requests to proceed
	ValidationBehaviourLog ValidationBehaviour = "log"
	// ValidationBehaviourIgnore ignores validation issues
	ValidationBehaviourIgnore ValidationBehaviour = "ignore"
)

// ValidationConfig represents the validation settings for requests and responses
type ValidationConfig struct {
	Request  interface{} `yaml:"request"`
	Response interface{} `yaml:"response"`
}

// GetRequestBehaviour returns the behaviour for request validation
func (c *ValidationConfig) GetRequestBehaviour() ValidationBehaviour {
	if c == nil {
		return getDefaultValidationBehaviour()
	}

	switch v := c.Request.(type) {
	case string:
		switch v {
		case "fail", "true":
			return ValidationBehaviourFail
		case "log":
			return ValidationBehaviourLog
		case "ignore", "false":
			return ValidationBehaviourIgnore
		default:
			return getDefaultValidationBehaviour()
		}
	case bool:
		if v {
			return ValidationBehaviourFail
		}
		return ValidationBehaviourIgnore
	default:
		return getDefaultValidationBehaviour()
	}
}

// GetResponseBehaviour returns the behaviour for response validation
func (c *ValidationConfig) GetResponseBehaviour() ValidationBehaviour {
	if c == nil {
		return getDefaultValidationBehaviour()
	}

	switch v := c.Response.(type) {
	case string:
		switch v {
		case "fail", "true":
			return ValidationBehaviourFail
		case "log":
			return ValidationBehaviourLog
		case "ignore", "false":
			return ValidationBehaviourIgnore
		default:
			return getDefaultValidationBehaviour()
		}
	case bool:
		if v {
			return ValidationBehaviourFail
		}
		return ValidationBehaviourIgnore
	default:
		return getDefaultValidationBehaviour()
	}
}

// IsRequestValidationEnabled returns true if request validation is enabled
func (c *ValidationConfig) IsRequestValidationEnabled() bool {
	behaviour := c.GetRequestBehaviour()
	return behaviour == ValidationBehaviourFail || behaviour == ValidationBehaviourLog
}

// IsResponseValidationEnabled returns true if response validation is enabled
func (c *ValidationConfig) IsResponseValidationEnabled() bool {
	behaviour := c.GetResponseBehaviour()
	return behaviour == ValidationBehaviourFail || behaviour == ValidationBehaviourLog
}

// getDefaultValidationBehaviour returns the default validation behaviour
// from the IMPOSTER_OPENAPI_VALIDATION_DEFAULT_BEHAVIOUR environment variable
func getDefaultValidationBehaviour() ValidationBehaviour {
	behaviour := os.Getenv("IMPOSTER_OPENAPI_VALIDATION_DEFAULT_BEHAVIOUR")
	switch behaviour {
	case "fail", "true":
		return ValidationBehaviourFail
	case "log":
		return ValidationBehaviourLog
	default:
		return ValidationBehaviourIgnore
	}
}

// Config represents the configuration for an Imposter mock server
type Config struct {
	Plugin       string          `yaml:"plugin"`
	BasePath     string          `yaml:"basePath,omitempty"`
	Resources    []Resource      `yaml:"resources,omitempty"`
	Interceptors []Interceptor   `yaml:"interceptors"`
	System       *System         `yaml:"system,omitempty"`
	Security     *SecurityConfig `yaml:"security"`
	Cors         *CorsConfig     `yaml:"cors,omitempty"`

	// SOAP-specific fields
	WSDLFile string `yaml:"wsdlFile,omitempty"`

	// OpenAPI-specific fields
	SpecFile        string            `yaml:"specFile,omitempty"`
	StripServerPath bool              `yaml:"stripServerPath,omitempty"`
	Validation      *ValidationConfig `yaml:"validation,omitempty"`
}

// ImposterConfig holds application-wide configuration
type ImposterConfig struct {
	LegacyConfigSupported bool
	ServerPort            string
	ServerUrl             string
}

// MatcherUnmarshaler is a helper type for unmarshaling Matcher from YAML
type MatcherUnmarshaler struct {
	Matcher Matcher
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for MatcherUnmarshaler
func (mu *MatcherUnmarshaler) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First try to unmarshal as a simple string
	var str string
	if err := unmarshal(&str); err == nil {
		mu.Matcher = StringMatcher(str)
		return nil
	}

	// If that fails, try to unmarshal as a MatchCondition
	var mc MatchCondition
	if err := unmarshal(&mc); err == nil {
		mu.Matcher = mc
		return nil
	}

	return fmt.Errorf("failed to unmarshal as either string or MatchCondition")
}
