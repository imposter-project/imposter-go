package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/imposter-project/imposter-go/internal/query"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/utils"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/exp/rand"
)

// ProcessTemplate processes a template string, replacing placeholders with actual values.
func ProcessTemplate(template string, r *http.Request, imposterConfig *config.ImposterConfig, requestStore *store.Store) string {
	// Replace request method
	template = strings.ReplaceAll(template, "${context.request.method}", r.Method)

	// Replace request path parameters
	for key, value := range r.URL.Query() {
		placeholder := fmt.Sprintf("${context.request.queryParams.%s}", key)
		template = replaceOrUseDefault(template, placeholder, func(plainExpr string) string {
			return value[0]
		})
	}

	// Replace request headers
	for key, value := range r.Header {
		placeholder := fmt.Sprintf("${context.request.headers.%s}", key)
		template = replaceOrUseDefault(template, placeholder, func(plainExpr string) string {
			return value[0]
		})
	}

	// Replace form parameters
	if err := r.ParseForm(); err == nil {
		for key, values := range r.Form {
			placeholder := fmt.Sprintf("${context.request.formParams.%s}", key)
			template = replaceOrUseDefault(template, placeholder, func(plainExpr string) string {
				return values[0]
			})
		}
	}

	// Replace path parameters
	pathParams := utils.ExtractPathParams(r.URL.Path, r.URL.Path)
	for key, value := range pathParams {
		placeholder := fmt.Sprintf("${context.request.pathParams.%s}", key)
		template = replaceOrUseDefault(template, placeholder, func(plainExpr string) string {
			return value
		})
	}

	// Replace request body
	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	template = strings.ReplaceAll(template, "${context.request.body}", string(body))

	// Replace request path
	template = strings.ReplaceAll(template, "${context.request.path}", r.URL.Path)

	// Replace request URI
	template = strings.ReplaceAll(template, "${context.request.uri}", r.URL.String())

	// Replace datetime placeholders
	now := time.Now()
	template = strings.ReplaceAll(template, "${datetime.now.iso8601_date}", now.Format("2006-01-02"))
	template = strings.ReplaceAll(template, "${datetime.now.iso8601_datetime}", now.Format(time.RFC3339))
	template = strings.ReplaceAll(template, "${datetime.now.millis}", fmt.Sprintf("%d", now.UnixMilli()))
	template = strings.ReplaceAll(template, "${datetime.now.nanos}", fmt.Sprintf("%d", now.UnixNano()))

	// Replace random placeholders
	template = replaceRandomPlaceholders(template)

	// Replace system/server placeholders
	template = strings.ReplaceAll(template, "${system.server.port}", imposterConfig.ServerPort)
	template = strings.ReplaceAll(template, "${system.server.url}", fmt.Sprintf("http://%s%s", r.Host, r.URL.Path))

	template = replaceStorePlaceholders(template, requestStore)
	return template
}

// replaceRandomPlaceholders handles random value placeholders.
func replaceRandomPlaceholders(template string) string {
	re := regexp.MustCompile(`\$\{random\.(\w+)\(([^)]*)\)\}`)
	return re.ReplaceAllStringFunc(template, func(match string) string {
		groups := re.FindStringSubmatch(match)
		function := groups[1]
		params := parseParams(groups[2])

		switch function {
		case "alphabetic":
			length := getIntParam(params, "length", 1)
			uppercase := getBoolParam(params, "uppercase", false)
			return randomAlphabetic(length, uppercase)
		case "alphanumeric":
			length := getIntParam(params, "length", 1)
			uppercase := getBoolParam(params, "uppercase", false)
			return randomAlphanumeric(length, uppercase)
		case "any":
			chars := getStringParam(params, "chars", "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
			length := getIntParam(params, "length", 1)
			uppercase := getBoolParam(params, "uppercase", false)
			return randomAny(chars, length, uppercase)
		case "numeric":
			length := getIntParam(params, "length", 1)
			return randomNumeric(length)
		case "uuid":
			uppercase := getBoolParam(params, "uppercase", false)
			return randomUUID(uppercase)
		default:
			return match
		}
	})
}

// randomUUID generates a random UUID string.
func randomUUID(uppercase bool) string {
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

// randomAlphabetic generates a random alphabetic string.
func randomAlphabetic(length int, uppercase bool) string {
	letters := "abcdefghijklmnopqrstuvwxyz"
	if uppercase {
		letters = strings.ToUpper(letters)
	}
	return randomStringFromCharset(length, letters)
}

// randomAlphanumeric generates a random alphanumeric string.
func randomAlphanumeric(length int, uppercase bool) string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if uppercase {
		letters = strings.ToUpper(letters)
	}
	return randomStringFromCharset(length, letters)
}

// randomAny generates a random string from a given character set.
func randomAny(chars string, length int, uppercase bool) string {
	if uppercase {
		chars = strings.ToUpper(chars)
	}
	return randomStringFromCharset(length, chars)
}

// randomNumeric generates a random numeric string.
func randomNumeric(length int) string {
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

// replaceStorePlaceholders replaces store placeholders in a template with actual values.
func replaceStorePlaceholders(tmpl string, requestStore *store.Store) string {
	re := regexp.MustCompile(`\$\{stores\.([^\.]+)\.([^}]+)\}`)
	return re.ReplaceAllStringFunc(tmpl, func(match string) string {
		return replaceOrUseDefault(match, match, func(plainExpr string) string {
			groups := re.FindStringSubmatch(plainExpr)
			storeName := groups[1]
			key := groups[2]
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
		})
	})
}

// getStoreValue retrieves a value from a store, or from the request store if the store is "request".
func getStoreValue(storeName, key string, requestStore *store.Store) (interface{}, bool) {
	if storeName == "request" {
		val, found := (*requestStore)[key]
		return val, found
	}
	return store.GetValue(storeName, key)
}

// extractTrailerFromExpr extracts the trailer from an expression, where
// the expression is of the form ${expr:trailer}.
// Both the trailer and the bare expression, without ${} or the trailer, are returned.
func extractTrailerFromExpr(expr string) (defaultVal string, plainExpr string) {
	inner := strings.Trim(expr, "${}")
	parts := strings.Split(inner, ":")
	if len(parts) == 2 {
		return parts[1], "${" + parts[0] + "}"
	}
	return "", expr
}

// replaceOrUseDefault replaces an expression in a template with a value, or a default value if the value is empty.
func replaceOrUseDefault(template string, expr string, repl func(plainExpr string) string) string {
	trailer, plainExpr := extractTrailerFromExpr(expr)
	actualVal := repl(plainExpr)

	var replacement string
	if actualVal == "" {
		// trailer was a default value in the form ${expr:-default}
		if strings.HasPrefix(trailer, "-") {
			replacement = strings.TrimPrefix("-", trailer)
		}
	} else {
		replacement = actualVal
	}

	// process query expressions
	if strings.HasPrefix(trailer, "$") {
		// jsonPath was provided in the form ${expr:jsonPath}
		result, _ := query.JsonPathQuery([]byte(actualVal), trailer)
		if result != nil {
			replacement = fmt.Sprintf("%v", result)
		}
	} else if strings.HasPrefix(trailer, "/") {
		// xPath was provided in the form ${expr:xPath}
		result, _ := query.XPathQuery([]byte(actualVal), trailer, nil)
		replacement = result
	}
	return strings.ReplaceAll(template, expr, replacement)
}
