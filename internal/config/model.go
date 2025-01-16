package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Response represents an HTTP response
type Response struct {
	Content    string            `yaml:"content"`
	StatusCode int               `yaml:"statusCode"`
	File       string            `yaml:"file"`
	Fail       string            `yaml:"fail"`
	Delay      Delay             `yaml:"delay"`
	Headers    map[string]string `yaml:"headers"`
	Template   bool              `yaml:"template"`
	SoapFault  bool              `yaml:"soapFault"`
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
		matched, _ := regexp.MatchString(m.Value, actualValue)
		return matched
	case "NotMatches":
		matched, _ := regexp.MatchString(m.Value, actualValue)
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

	// Capture request data - TODO move to a separate struct
	Capture map[string]Capture `yaml:"capture,omitempty"`

	// SOAP-specific fields
	Operation  string `yaml:"operation,omitempty"`
	SOAPAction string `yaml:"soapAction,omitempty"`
	Binding    string `yaml:"binding,omitempty"`
}

// Resource represents an HTTP resource
type Resource struct {
	RequestMatcher `yaml:",inline"`
	Response       Response        `yaml:"response"`
	Security       *SecurityConfig `yaml:"security,omitempty"`
}

// Interceptor represents an HTTP interceptor that can be executed before resources
type Interceptor struct {
	RequestMatcher `yaml:",inline"`
	Response       *Response `yaml:"response,omitempty"`
	Continue       bool      `yaml:"continue"`
}

type System struct {
	Stores        map[string]StoreDefinition `yaml:"stores"`
	XMLNamespaces map[string]string          `yaml:"xmlNamespaces,omitempty"`
}

type StoreDefinition struct {
	PreloadFile string                 `yaml:"preloadFile,omitempty"`
	PreloadData map[string]interface{} `yaml:"preloadData,omitempty"`
}

// Config represents the configuration for a mock service
type Config struct {
	Plugin       string          `yaml:"plugin"`
	BasePath     string          `yaml:"basePath"`
	Resources    []Resource      `yaml:"resources"`
	Interceptors []Interceptor   `yaml:"interceptors"`
	System       *System         `yaml:"system"`
	Security     *SecurityConfig `yaml:"security"`

	// SOAP-specific fields
	WSDLFile string `yaml:"wsdlFile,omitempty"`

	// OpenAPI-specific fields
	SpecFile string `yaml:"specFile,omitempty"`
}

// ImposterConfig holds application-wide configuration
type ImposterConfig struct {
	ServerPort string
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
