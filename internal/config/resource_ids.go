package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// PreCalculateResourceIDs pre-calculates and stores resource IDs for all resources
// This should be called once after loading all configuration files to avoid repeated computation
func PreCalculateResourceIDs(configs []Config) {
	for configIdx := range configs {
		cfg := &configs[configIdx]

		// Process resources
		for resIdx := range cfg.Resources {
			res := &cfg.Resources[resIdx]
			method := res.Method
			var name string

			// Determine resource name based on plugin type
			switch cfg.Plugin {
			case "soap":
				name = res.Operation
			default: // rest and other plugins
				name = res.Path
			}

			res.ResourceID = GenerateResourceKey(method, name, &res.RequestMatcher)
		}

		// Process interceptors
		for intIdx := range cfg.Interceptors {
			interceptor := &cfg.Interceptors[intIdx]
			method := interceptor.Method
			var name string

			// Determine resource name based on plugin type
			switch cfg.Plugin {
			case "soap":
				name = interceptor.Operation
			default: // rest and other plugins
				name = interceptor.Path
			}

			interceptor.ResourceID = GenerateResourceKey(method, name, &interceptor.RequestMatcher)
		}
	}
}

// GenerateResourceKey generates a unique key for a resource including all matching criteria
func GenerateResourceKey(method, name string, matcher *RequestMatcher) string {
	if method == "" {
		method = "*"
	}
	if name == "" {
		name = "*"
	}
	baseKey := fmt.Sprintf("%s:%s", strings.ToUpper(method), name)

	// If no additional matching criteria, use the simple key
	if matcher == nil || isEmptyMatcher(matcher) {
		return baseKey
	}

	// Generate hash of all matching criteria
	hash := generateMatcherHash(matcher)
	return fmt.Sprintf("%s:%s", baseKey, hash)
}

// isEmptyMatcher checks if the matcher has no additional criteria beyond method/path/operation
func isEmptyMatcher(matcher *RequestMatcher) bool {
	return len(matcher.RequestHeaders) == 0 &&
		len(matcher.QueryParams) == 0 &&
		len(matcher.FormParams) == 0 &&
		len(matcher.PathParams) == 0 &&
		matcher.RequestBody.BodyMatchCondition == nil &&
		len(matcher.RequestBody.AllOf) == 0 &&
		len(matcher.RequestBody.AnyOf) == 0 &&
		len(matcher.AllOf) == 0 &&
		len(matcher.AnyOf) == 0 &&
		matcher.SOAPAction == "" &&
		matcher.Binding == ""
}

// generateMatcherHash creates a deterministic hash of the matcher's criteria
func generateMatcherHash(matcher *RequestMatcher) string {
	h := sha256.New()

	// Collect all matching criteria in sorted order for deterministic hashing
	var parts []string

	// RequestHeaders
	if len(matcher.RequestHeaders) > 0 {
		headerKeys := make([]string, 0, len(matcher.RequestHeaders))
		for k := range matcher.RequestHeaders {
			headerKeys = append(headerKeys, k)
		}
		sort.Strings(headerKeys)
		for _, k := range headerKeys {
			parts = append(parts, fmt.Sprintf("header:%s=%v", k, matcher.RequestHeaders[k]))
		}
	}

	// QueryParams
	if len(matcher.QueryParams) > 0 {
		queryKeys := make([]string, 0, len(matcher.QueryParams))
		for k := range matcher.QueryParams {
			queryKeys = append(queryKeys, k)
		}
		sort.Strings(queryKeys)
		for _, k := range queryKeys {
			parts = append(parts, fmt.Sprintf("query:%s=%v", k, matcher.QueryParams[k]))
		}
	}

	// FormParams
	if len(matcher.FormParams) > 0 {
		formKeys := make([]string, 0, len(matcher.FormParams))
		for k := range matcher.FormParams {
			formKeys = append(formKeys, k)
		}
		sort.Strings(formKeys)
		for _, k := range formKeys {
			parts = append(parts, fmt.Sprintf("form:%s=%v", k, matcher.FormParams[k]))
		}
	}

	// PathParams
	if len(matcher.PathParams) > 0 {
		pathKeys := make([]string, 0, len(matcher.PathParams))
		for k := range matcher.PathParams {
			pathKeys = append(pathKeys, k)
		}
		sort.Strings(pathKeys)
		for _, k := range pathKeys {
			parts = append(parts, fmt.Sprintf("path:%s=%v", k, matcher.PathParams[k]))
		}
	}

	// RequestBody
	if matcher.RequestBody.BodyMatchCondition != nil {
		parts = append(parts, fmt.Sprintf("body:value=%s,operator=%s,jsonpath=%s,xpath=%s",
			matcher.RequestBody.BodyMatchCondition.Value,
			matcher.RequestBody.BodyMatchCondition.Operator,
			matcher.RequestBody.BodyMatchCondition.JSONPath,
			matcher.RequestBody.BodyMatchCondition.XPath))
	}
	if len(matcher.RequestBody.AllOf) > 0 {
		parts = append(parts, fmt.Sprintf("body:allof=%v", matcher.RequestBody.AllOf))
	}
	if len(matcher.RequestBody.AnyOf) > 0 {
		parts = append(parts, fmt.Sprintf("body:anyof=%v", matcher.RequestBody.AnyOf))
	}

	// Expression conditions
	if len(matcher.AllOf) > 0 {
		parts = append(parts, fmt.Sprintf("allof=%v", matcher.AllOf))
	}
	if len(matcher.AnyOf) > 0 {
		parts = append(parts, fmt.Sprintf("anyof=%v", matcher.AnyOf))
	}

	// SOAP-specific fields
	if matcher.SOAPAction != "" {
		parts = append(parts, fmt.Sprintf("soapaction=%s", matcher.SOAPAction))
	}
	if matcher.Binding != "" {
		parts = append(parts, fmt.Sprintf("binding=%s", matcher.Binding))
	}

	// Sort all parts to ensure deterministic order
	sort.Strings(parts)

	// Write to hash
	for _, part := range parts {
		h.Write([]byte(part))
	}

	// Return first 8 characters of hex-encoded hash for reasonable key length
	fullHash := hex.EncodeToString(h.Sum(nil))
	return fullHash[:8]
}
