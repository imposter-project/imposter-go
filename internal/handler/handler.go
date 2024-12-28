package handler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"regexp"

	"encoding/json"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/gatehill/imposter-go/internal/config"
	"github.com/gatehill/imposter-go/internal/matcher"
	"github.com/gatehill/imposter-go/internal/store"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/exp/rand"
	"k8s.io/client-go/util/jsonpath"
)

// HandleRequest processes incoming HTTP requests based on resources
func HandleRequest(w http.ResponseWriter, r *http.Request, configDir string, configs []config.Config, imposterConfig *config.ImposterConfig) {
	body, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewReader(body))

	type matchResult struct {
		Resource config.Resource
		Score    int
	}

	var allMatches []matchResult

	for _, cfg := range configs {
		for _, res := range cfg.Resources {
			score := calculateMatchScore(res, r, body)
			if score > 0 {
				allMatches = append(allMatches, matchResult{Resource: res, Score: score})
			}
		}
	}

	if len(allMatches) == 0 {
		notFoundMsg := "Resource not found"
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, notFoundMsg)
		fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
			r.Method, r.URL.Path, http.StatusNotFound, len(notFoundMsg))
		return
	}

	// Find the match with the highest score; track if there's a tie
	best := allMatches[0]
	tie := false
	for _, m := range allMatches[1:] {
		if m.Score > best.Score {
			best = m
			tie = false
		} else if m.Score == best.Score {
			tie = true
		}
	}

	if tie {
		fmt.Printf("Warning: multiple equally specific matches. Using the first.\n")
	}

	// Capture request data
	captureRequestData(imposterConfig, best.Resource, r, body)

	// Handle delay if specified
	if best.Resource.Response.Delay.Exact > 0 {
		delay := best.Resource.Response.Delay.Exact
		fmt.Printf("Delaying request (exact: %dms) - method:%s, path:%s\n", delay, r.Method, r.URL.Path)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	} else if best.Resource.Response.Delay.Min > 0 && best.Resource.Response.Delay.Max > 0 {
		delay := rand.Intn(best.Resource.Response.Delay.Max-best.Resource.Response.Delay.Min+1) + best.Resource.Response.Delay.Min
		fmt.Printf("Delaying request (range: %dms-%dms, actual: %dms) - method:%s, path:%s\n",
			best.Resource.Response.Delay.Min, best.Resource.Response.Delay.Max, delay, r.Method, r.URL.Path)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	// Write response using 'best.Resource'
	statusCode := best.Resource.Response.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}

	// Set response headers
	for key, value := range best.Resource.Response.Headers {
		w.Header().Set(key, value)
	}

	var responseContent string
	if best.Resource.Response.File != "" {
		filePath := filepath.Join(configDir, best.Resource.Response.File)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
		responseContent = string(data)
	} else {
		responseContent = best.Resource.Response.Content
	}

	if best.Resource.Response.Template {
		responseContent = processTemplate(responseContent, r, imposterConfig)
	}

	if best.Resource.Response.Fail != "" {
		switch best.Resource.Response.Fail {
		case "EmptyResponse":
			// Send a status but no body
			w.WriteHeader(statusCode)
			fmt.Printf("Handled request (simulated failure: EmptyResponse) - method:%s, path:%s, status:%d, length:0\n",
				r.Method, r.URL.Path, statusCode)
			return

		case "CloseConnection":
			// Close the connection before sending any response
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "HTTP server does not support connection hijacking", http.StatusInternalServerError)
				return
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
				return
			}
			fmt.Printf("Handled request (simulated failure: CloseConnection) - method:%s, path:%s\n", r.Method, r.URL.Path)
			conn.Close()
			return
		}
	}

	w.Write([]byte(responseContent))
	fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
		r.Method, r.URL.Path, statusCode, len(responseContent))
}

func captureRequestData(imposterConfig *config.ImposterConfig, resource config.Resource, r *http.Request, body []byte) {
	for key, capture := range resource.Capture {
		var value string
		if capture.PathParam != "" {
			value = extractPathParams(r.URL.Path, resource.Path)[capture.PathParam]
		} else if capture.QueryParam != "" {
			value = r.URL.Query().Get(capture.QueryParam)
		} else if capture.FormParam != "" {
			if err := r.ParseForm(); err == nil {
				value = r.FormValue(capture.FormParam)
			}
		} else if capture.RequestHeader != "" {
			value = r.Header.Get(capture.RequestHeader)
		} else if capture.Expression != "" {
			value = processTemplate(capture.Expression, r, imposterConfig)
		} else if capture.Const != "" {
			value = capture.Const
		} else if capture.RequestBody.JSONPath != "" {
			value = extractJSONPath(body, capture.RequestBody.JSONPath)
		} else if capture.RequestBody.XPath != "" {
			value = extractXPath(body, capture.RequestBody.XPath, capture.RequestBody.XMLNamespaces)
		}
		if value != "" {
			store.StoreValue(capture.Store, key, value)
		}
	}
}

func extractJSONPath(body []byte, jsonPath string) string {
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return ""
	}

	jpath := jsonpath.New("jsonpath")
	if err := jpath.Parse(jsonPath); err != nil {
		return ""
	}

	results := new(bytes.Buffer)
	if err := jpath.Execute(results, jsonData); err != nil {
		return ""
	}

	return results.String()
}

func extractXPath(body []byte, xPath string, namespaces map[string]string) string {
	doc, err := xmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		return ""
	}

	expr, err := xpath.CompileWithNS(xPath, namespaces)
	if err != nil {
		return ""
	}

	result := xmlquery.QuerySelector(doc, expr)
	if result == nil {
		return ""
	}

	return result.InnerText()
}

// processTemplate replaces placeholders in the template with actual values
func processTemplate(template string, r *http.Request, imposterConfig *config.ImposterConfig) string {
	// Replace request path parameters
	for key, value := range r.URL.Query() {
		placeholder := fmt.Sprintf("${context.request.queryParams.%s}", key)
		template = strings.ReplaceAll(template, placeholder, value[0])
	}

	// Replace request headers
	for key, value := range r.Header {
		placeholder := fmt.Sprintf("${context.request.headers.%s}", key)
		template = strings.ReplaceAll(template, placeholder, value[0])
	}

	// Replace form parameters
	if err := r.ParseForm(); err == nil {
		for key, values := range r.Form {
			placeholder := fmt.Sprintf("${context.request.formParams.%s}", key)
			template = strings.ReplaceAll(template, placeholder, values[0])
		}
	}

	// Replace request body
	body, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewReader(body))
	template = strings.ReplaceAll(template, "${context.request.body}", string(body))

	// Replace request path
	template = strings.ReplaceAll(template, "${context.request.path}", r.URL.Path)

	// Replace request URI
	template = strings.ReplaceAll(template, "${context.request.uri}", r.URL.String())

	// Replace path parameters
	pathParams := extractPathParams(r.URL.Path, r.URL.Path)
	for key, value := range pathParams {
		placeholder := fmt.Sprintf("${context.request.pathParams.%s}", key)
		template = strings.ReplaceAll(template, placeholder, value)
	}

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

	template = replaceStorePlaceholders(template)
	return template
}

// extractPathParams extracts path parameters from the request path
func extractPathParams(requestPath, resourcePath string) map[string]string {
	requestSegments := strings.Split(strings.Trim(requestPath, "/"), "/")
	resourceSegments := strings.Split(strings.Trim(resourcePath, "/"), "/")
	pathParams := make(map[string]string)

	for i, segment := range resourceSegments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			paramName := strings.Trim(segment, "{}")
			pathParams[paramName] = requestSegments[i]
		}
	}

	return pathParams
}

// replaceRandomPlaceholders handles random value placeholders
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

func randomUUID(uppercase bool) string {
	uuidStr := uuid.NewV4().String()
	if uppercase {
		return strings.ToUpper(uuidStr)
	}
	return uuidStr
}

// Helper functions for random value generation and parameter parsing

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

func getIntParam(params map[string]string, key string, defaultValue int) int {
	if value, exists := params[key]; exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolParam(params map[string]string, key string, defaultValue bool) bool {
	if value, exists := params[key]; exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getStringParam(params map[string]string, key string, defaultValue string) string {
	if value, exists := params[key]; exists {
		return value
	}
	return defaultValue
}

func randomAlphabetic(length int, uppercase bool) string {
	letters := "abcdefghijklmnopqrstuvwxyz"
	if uppercase {
		letters = strings.ToUpper(letters)
	}
	return randomStringFromCharset(length, letters)
}

func randomAlphanumeric(length int, uppercase bool) string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if uppercase {
		letters = strings.ToUpper(letters)
	}
	return randomStringFromCharset(length, letters)
}

func randomAny(chars string, length int, uppercase bool) string {
	if uppercase {
		chars = strings.ToUpper(chars)
	}
	return randomStringFromCharset(length, chars)
}

func randomNumeric(length int) string {
	digits := "0123456789"
	return randomStringFromCharset(length, digits)
}

func randomStringFromCharset(length int, charset string) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// ...existing code...

// calculateMatchScore returns the number of matched constraints.
// Returns 0 if any required condition fails, meaning no match.
func calculateMatchScore(res config.Resource, r *http.Request, body []byte) int {
	score := 0

	// Match method
	if r.Method != res.Method {
		return 0
	}
	score++

	// Match path with optional pathParams
	requestSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	resourceSegments := strings.Split(strings.Trim(res.Path, "/"), "/")
	if len(requestSegments) != len(resourceSegments) {
		return 0
	}

	for i, segment := range resourceSegments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			paramName := strings.Trim(segment, "{}")
			if condition, hasParam := res.PathParams[paramName]; hasParam {
				if !matcher.MatchSimpleOrAdvancedCondition(requestSegments[i], condition) {
					return 0
				}
				score++
			}
		} else {
			if requestSegments[i] != segment {
				return 0
			}
		}
	}

	// Match query parameters
	for key, condition := range res.QueryParams {
		actualValue := r.URL.Query().Get(key)
		if !matcher.MatchSimpleOrAdvancedCondition(actualValue, condition) {
			return 0
		}
		score++
	}

	// Match headers
	for key, condition := range res.Headers {
		actualValue := r.Header.Get(key)
		if !matcher.MatchSimpleOrAdvancedCondition(actualValue, condition) {
			return 0
		}
		score++
	}

	// Match form parameters (if content type is application/x-www-form-urlencoded)
	if len(res.FormParams) > 0 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return 0
		}
		for key, condition := range res.FormParams {
			if !matcher.MatchSimpleOrAdvancedCondition(r.FormValue(key), condition) {
				return 0
			}
			score++
		}
	}

	// Match request body
	if res.RequestBody.JSONPath != "" {
		if !matcher.MatchJSONPath(body, res.RequestBody.BodyMatchCondition) {
			return 0
		}
		score++
	} else if res.RequestBody.XPath != "" {
		if !matcher.MatchXPath(body, res.RequestBody.BodyMatchCondition) {
			return 0
		}
		score++
	} else if res.RequestBody.Value != "" {
		if !matcher.MatchCondition(string(body), res.RequestBody.MatchCondition) {
			return 0
		}
		score++
	} else if len(res.RequestBody.AllOf) > 0 {
		for _, condition := range res.RequestBody.AllOf {
			if condition.JSONPath != "" {
				if !matcher.MatchJSONPath(body, condition) {
					return 0
				}
			} else if condition.XPath != "" {
				if !matcher.MatchXPath(body, condition) {
					return 0
				}
			} else if !matcher.MatchCondition(string(body), condition.MatchCondition) {
				return 0
			}
		}
		score++
	} else if len(res.RequestBody.AnyOf) > 0 {
		matched := false
		for _, condition := range res.RequestBody.AnyOf {
			if condition.JSONPath != "" {
				if matcher.MatchJSONPath(body, condition) {
					matched = true
					break
				}
			} else if condition.XPath != "" {
				if matcher.MatchXPath(body, condition) {
					matched = true
					break
				}
			} else if matcher.MatchCondition(string(body), condition.MatchCondition) {
				matched = true
				break
			}
		}
		if !matched {
			return 0
		}
		score++
	}

	return score
}

func replaceStorePlaceholders(tmpl string) string {
	re := regexp.MustCompile(`\$\{stores\.([^\.]+)\.([^}]+)\}`)
	return re.ReplaceAllStringFunc(tmpl, func(match string) string {
		groups := re.FindStringSubmatch(match)
		storeName := groups[1]
		key := groups[2]
		val, found := store.GetValue(storeName, key)
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
}
