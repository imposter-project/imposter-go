package template

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/query"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/utils"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/exp/rand"
)

// ProcessTemplate processes a template string using the provided context.
// This is the new version that should be used going forward.
func ProcessTemplate(template string, exch *exchange.Exchange, imposterConfig *config.ImposterConfig, reqMatcher *config.RequestMatcher) string {
	if template == "" {
		return ""
	}

	// Handle both formats:
	// 1. ${category.subcategory.field}
	// 2. ${category.function(param=value,...)}
	re := regexp.MustCompile(`\$\{([^\.]+)\.([^\.}(]+)(?:\.([^}]+)|(?:\(([^)]*)\)))\}`)
	return re.ReplaceAllStringFunc(template, func(match string) string {
		groups := re.FindStringSubmatch(match)
		if len(groups) < 5 {
			return match
		}

		category := groups[1]
		subcategory := groups[2]
		field := groups[3]      // For dot notation
		parameters := groups[4] // For function call notation

		// Handle random functions differently
		if category == "random" {
			return handleRandomReplacement(subcategory, parameters)
		}

		// Extract any trailer (default value or query expression) for dot notation
		var trailer string
		if field != "" {
			trailer, field = extractTrailer(field)
		}

		// Get the raw value based on category
		var rawValue string
		switch category {
		case "context":
			rawValue = handleContextReplacement(subcategory, field, reqMatcher, exch)
		case "stores":
			rawValue = handleStoreReplacement(subcategory, field, exch.RequestStore)
		case "datetime":
			rawValue = handleDatetimeReplacement(subcategory, field)
		case "system":
			rawValue = handleSystemReplacement(subcategory, field, exch, imposterConfig)
		default:
			return match
		}

		// Process the raw value with any trailer
		return processWithTrailer(rawValue, trailer)
	})
}

// extractTrailer extracts any trailer (default value or query expression) from a field
func extractTrailer(field string) (trailer, cleanField string) {
	parts := strings.SplitN(field, ":", 2)
	if len(parts) == 2 {
		return parts[1], parts[0]
	}
	return "", field
}

// processWithTrailer processes a value with a trailer (default value or query expression)
func processWithTrailer(value, trailer string) string {
	if value == "" {
		// Handle default value
		if strings.HasPrefix(trailer, "-") {
			return strings.TrimPrefix(trailer, "-")
		}
		return ""
	}

	// Handle query expressions
	if strings.HasPrefix(trailer, "$") {
		// JSONPath expression
		result, ok := query.JsonPathQuery([]byte(value), trailer)
		if ok && result != nil {
			return fmt.Sprintf("%v", result)
		}
		return ""
	} else if strings.HasPrefix(trailer, "/") {
		// XPath expression
		result, ok := query.XPathQuery([]byte(value), trailer, nil)
		if ok {
			return result
		}
		return ""
	}

	return value
}

// handleContextReplacement handles replacements for context.request.* and context.response.*
func handleContextReplacement(subcategory, field string, reqMatcher *config.RequestMatcher, exch *exchange.Exchange) string {
	switch subcategory {
	case "request":
		return handleRequestReplacement(field, reqMatcher, exch)
	case "response":
		return handleResponseReplacement(field, exch)
	default:
		return ""
	}
}

// handleRequestReplacement handles replacements for context.request.*
func handleRequestReplacement(field string, reqMatcher *config.RequestMatcher, exch *exchange.Exchange) string {
	switch {
	case field == "method":
		return exch.Request.Request.Method
	case field == "path":
		return exch.Request.Request.URL.Path
	case field == "uri":
		return exch.Request.Request.URL.String()
	case field == "body":
		return string(exch.Request.Body)
	case strings.HasPrefix(field, "queryParams."):
		key := strings.TrimPrefix(field, "queryParams.")
		return exch.Request.Request.URL.Query().Get(key)
	case strings.HasPrefix(field, "headers."):
		key := strings.TrimPrefix(field, "headers.")
		return exch.Request.Request.Header.Get(key)
	case strings.HasPrefix(field, "pathParams."):
		key := strings.TrimPrefix(field, "pathParams.")
		params := utils.ExtractPathParams(exch.Request.Request.URL.Path, reqMatcher.Path)
		return params[key]
	case strings.HasPrefix(field, "formParams."):
		key := strings.TrimPrefix(field, "formParams.")
		_ = exch.Request.Request.ParseForm()
		return exch.Request.Request.FormValue(key)
	default:
		return ""
	}
}

// handleResponseReplacement handles replacements for context.response.*
func handleResponseReplacement(field string, exch *exchange.Exchange) string {
	if exch.Response == nil {
		return ""
	}

	switch {
	case field == "body":
		if exch.Response != nil {
			return string(exch.Response.Body)
		}
		return ""
	case field == "statusCode":
		if exch.Response != nil && exch.Response.Response != nil {
			return fmt.Sprintf("%d", exch.Response.Response.StatusCode)
		}
		return ""
	case strings.HasPrefix(field, "headers."):
		key := strings.TrimPrefix(field, "headers.")
		if exch.Response != nil && exch.Response.Response != nil {
			return exch.Response.Response.Header.Get(key)
		}
		return ""
	default:
		return ""
	}
}

// handleStoreReplacement handles replacements for stores.*
func handleStoreReplacement(storeName, key string, requestStore *store.Store) string {
	val, found := getStoreValue(storeName, key, requestStore)
	if !found {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(jsonBytes)
	}
}

// handleDatetimeReplacement handles replacements for datetime.*
func handleDatetimeReplacement(subcategory, field string) string {
	if subcategory != "now" {
		return ""
	}

	now := time.Now()
	switch field {
	case "iso8601_date":
		return now.Format("2006-01-02")
	case "iso8601_datetime":
		return now.Format(time.RFC3339)
	case "millis":
		return fmt.Sprintf("%d", now.UnixMilli())
	case "nanos":
		return fmt.Sprintf("%d", now.UnixNano())
	default:
		return ""
	}
}

// handleRandomReplacement handles replacements for random.*
func handleRandomReplacement(function, params string) string {
	// Extract parameters from the field string
	paramMap := parseParams(params)

	switch function {
	case "alphabetic":
		length := getIntParam(paramMap, "length", 1)
		uppercase := getBoolParam(paramMap, "uppercase", false)
		return RandomAlphabetic(length, uppercase)
	case "alphanumeric":
		length := getIntParam(paramMap, "length", 1)
		uppercase := getBoolParam(paramMap, "uppercase", false)
		return RandomAlphanumeric(length, uppercase)
	case "any":
		chars := getStringParam(paramMap, "chars", "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		length := getIntParam(paramMap, "length", 1)
		uppercase := getBoolParam(paramMap, "uppercase", false)
		return RandomAny(chars, length, uppercase)
	case "numeric":
		length := getIntParam(paramMap, "length", 1)
		return RandomNumeric(length)
	case "uuid":
		uppercase := getBoolParam(paramMap, "uppercase", false)
		return RandomUUID(uppercase)
	default:
		return ""
	}
}

// handleSystemReplacement handles replacements for system.*
func handleSystemReplacement(subcategory, field string, exch *exchange.Exchange, imposterConfig *config.ImposterConfig) string {
	if imposterConfig == nil {
		return ""
	}

	if subcategory != "server" {
		return ""
	}

	switch field {
	case "port":
		return imposterConfig.ServerPort
	case "url":
		return imposterConfig.ServerUrl
	default:
		return ""
	}
}

// RandomUUID generates a random UUID string.
func RandomUUID(uppercase bool) string {
	uuidStr := uuid.NewV4().String()
	if uppercase {
		return strings.ToUpper(uuidStr)
	}
	return uuidStr
}

// parseParams parses a parameter string into a map.
func parseParams(paramStr string) map[string]string {
	params := make(map[string]string)
	if paramStr == "" {
		return params
	}
	for _, param := range strings.Split(paramStr, ",") {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) == 2 {
			params[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return params
}

// getIntParam retrieves an integer parameter from a map, or returns a default value.
func getIntParam(params map[string]string, key string, defaultValue int) int {
	if value, exists := params[key]; exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getBoolParam retrieves a boolean parameter from a map, or returns a default value.
func getBoolParam(params map[string]string, key string, defaultValue bool) bool {
	if value, exists := params[key]; exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getStringParam retrieves a string parameter from a map, or returns a default value.
func getStringParam(params map[string]string, key string, defaultValue string) string {
	if value, exists := params[key]; exists {
		return value
	}
	return defaultValue
}

// RandomAlphabetic generates a random alphabetic string.
func RandomAlphabetic(length int, uppercase bool) string {
	letters := "abcdefghijklmnopqrstuvwxyz"
	if uppercase {
		letters = strings.ToUpper(letters)
	}
	return randomStringFromCharset(length, letters)
}

// RandomAlphanumeric generates a random alphanumeric string.
func RandomAlphanumeric(length int, uppercase bool) string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if uppercase {
		letters = strings.ToUpper(letters)
	}
	return randomStringFromCharset(length, letters)
}

// RandomAny generates a random string from a given character set.
func RandomAny(chars string, length int, uppercase bool) string {
	if uppercase {
		chars = strings.ToUpper(chars)
	}
	return randomStringFromCharset(length, chars)
}

// RandomNumeric generates a random numeric string.
func RandomNumeric(length int) string {
	digits := "0123456789"
	return randomStringFromCharset(length, digits)
}

// randomStringFromCharset generates a random string from a given character set.
func randomStringFromCharset(length int, charset string) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// getStoreValue retrieves a value from a store, or from the request store if the store is "request".
func getStoreValue(storeName, key string, requestStore *store.Store) (interface{}, bool) {
	s := store.Open(storeName, requestStore)
	return s.GetValue(key)
}
