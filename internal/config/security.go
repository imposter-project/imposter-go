package config

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

// resourceCounter is used to generate unique prefixes for resource-level security conditions
var resourceCounter uint64

// SecurityConfig represents the security configuration block
type SecurityConfig struct {
	Default    string              `yaml:"default"`
	Conditions []SecurityCondition `yaml:"conditions"`
}

// SecurityCondition represents a single security condition
type SecurityCondition struct {
	Effect         string                        `yaml:"effect"`
	QueryParams    map[string]MatcherUnmarshaler `yaml:"queryParams"`
	FormParams     map[string]MatcherUnmarshaler `yaml:"formParams"`
	RequestHeaders map[string]MatcherUnmarshaler `yaml:"requestHeaders"`
}

// transformSecurityConfig converts security configuration into interceptors
func transformSecurityConfig(cfg *Config) {
	var interceptors []Interceptor

	// Transform root-level security first
	if cfg.Security != nil {
		rootInterceptors := transformSecurityBlock(cfg, cfg.Security, "")
		interceptors = append(interceptors, *rootInterceptors...)
		cfg.Security = nil
	}

	// Transform resource-level security
	for i := range cfg.Resources {
		if cfg.Resources[i].Security != nil {
			// Generate a unique prefix for this resource
			prefix := fmt.Sprintf("resource%d_", atomic.AddUint64(&resourceCounter, 1))
			resourceInterceptors := transformSecurityBlock(cfg, cfg.Resources[i].Security, prefix)
			interceptors = append(interceptors, *resourceInterceptors...)
			cfg.Resources[i].Security = nil
		}
	}

	// always put security interceptors at the beginning
	cfg.Interceptors = append(interceptors, cfg.Interceptors...)
}

// transformSecurityBlock converts a security block into interceptors
// prefix is used to make condition keys unique across different security blocks
func transformSecurityBlock(cfg *Config, security *SecurityConfig, prefix string) *[]Interceptor {
	var interceptors []Interceptor

	// Create a map to store condition states
	for i, condition := range security.Conditions {
		// Create a unique key for this condition
		conditionKey := fmt.Sprintf("%ssecurity_condition%d", prefix, i+1)

		// Create an interceptor for the condition check
		interceptor := Interceptor{
			RequestMatcher: RequestMatcher{
				RequestHeaders: make(map[string]MatcherUnmarshaler),
				QueryParams:    make(map[string]MatcherUnmarshaler),
				FormParams:     make(map[string]MatcherUnmarshaler),
				Capture: map[string]Capture{
					conditionKey: {
						Store: "request",
						CaptureConfig: CaptureConfig{
							Const: "met",
						},
					},
				},
			},
			Continue: true,
		}

		// Add header conditions
		for header, matcher := range condition.RequestHeaders {
			interceptor.RequestHeaders[header] = matcher
		}

		// Add query parameter conditions
		for param, matcher := range condition.QueryParams {
			interceptor.QueryParams[param] = matcher
		}

		// Add form parameter conditions
		for param, matcher := range condition.FormParams {
			interceptor.FormParams[param] = matcher
		}

		interceptors = append(interceptors, interceptor)
	}

	// Add default deny interceptor if default is "Deny"
	if strings.EqualFold(security.Default, "Deny") {
		denyInterceptor := Interceptor{
			RequestMatcher: RequestMatcher{
				AnyOf: buildSecurityEvalConditions(len(security.Conditions), prefix),
			},
			Response: &Response{
				StatusCode: http.StatusUnauthorized,
				Content:    "Unauthorised",
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
			},
			Continue: false,
		}
		interceptors = append(interceptors, denyInterceptor)
	}

	return &interceptors
}

// buildSecurityEvalConditions creates evaluation conditions to check if any security condition was met
func buildSecurityEvalConditions(numConditions int, prefix string) []ExpressionMatchCondition {
	if numConditions == 0 {
		return []ExpressionMatchCondition{}
	}
	conditions := make([]ExpressionMatchCondition, 0, numConditions)
	for i := 1; i <= numConditions; i++ {
		conditions = append(conditions, ExpressionMatchCondition{
			Expression: fmt.Sprintf("${stores.request.%ssecurity_condition%d}", prefix, i),
			MatchCondition: MatchCondition{
				Value:    "met",
				Operator: "NotEqualTo",
			},
		})
	}
	return conditions
}
