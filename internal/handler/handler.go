package handler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gatehill/imposter-go/internal/matcher"
)

// Response defines the response structure
type Response struct {
	Content    string `yaml:"content"`
	StatusCode int    `yaml:"statusCode"`
	File       string `yaml:"file"`
}

// Resource defines a structure for mock HTTP responses
type Resource struct {
	Method      string            `yaml:"method"`
	Path        string            `yaml:"path"`
	QueryParams map[string]string `yaml:"queryParams"`
	Headers     map[string]string `yaml:"headers"`
	RequestBody map[string]string `yaml:"requestBody"`
	Response    Response          `yaml:"response"`
}

// HandleRequest processes incoming HTTP requests based on resources
func HandleRequest(w http.ResponseWriter, r *http.Request, configDir string, resources []Resource) {
	body, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewReader(body)) // Restore body for further reading

	for _, res := range resources {
		if matchesResource(res, r, body) {
			statusCode := res.Response.StatusCode
			if statusCode == 0 {
				statusCode = 200
			}
			w.WriteHeader(statusCode)

			if res.Response.File != "" {
				filePath := filepath.Join(configDir, res.Response.File)
				data, err := ioutil.ReadFile(filePath)
				if err != nil {
					http.Error(w, "Failed to read file", http.StatusInternalServerError)
					return
				}
				w.Write(data)
				return
			}

			if res.Response.Content != "" {
				fmt.Fprint(w, res.Response.Content)
				return
			}
		}
	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "Resource not found")
}

// matchesResource checks if an incoming HTTP request matches the defined resource
func matchesResource(res Resource, r *http.Request, body []byte) bool {
	// Match method and path
	if r.Method != res.Method || r.URL.Path != res.Path {
		return false
	}

	// Match query parameters
	for key, expectedValue := range res.QueryParams {
		actualValue := r.URL.Query().Get(key)
		if actualValue != expectedValue {
			return false
		}
	}

	// Match headers
	for key, expectedValue := range res.Headers {
		actualValue := r.Header.Get(key)
		if !strings.EqualFold(actualValue, expectedValue) {
			return false
		}
	}

	// Match request body
	if xpathQuery, exists := res.RequestBody["xpath"]; exists {
		if !matcher.MatchXPath(body, xpathQuery) {
			return false
		}
	}

	if jsonPathQuery, exists := res.RequestBody["jsonpath"]; exists {
		if !matcher.MatchJSONPath(body, jsonPathQuery) {
			return false
		}
	}

	return true
}