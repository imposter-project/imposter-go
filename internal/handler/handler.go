package handler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gatehill/imposter-go/internal/config"
	"github.com/gatehill/imposter-go/internal/matcher"
)

// HandleRequest processes incoming HTTP requests based on resources
func HandleRequest(w http.ResponseWriter, r *http.Request, configDir string, configs []config.Config) {
	body, _ := ioutil.ReadAll(r.Body)
	r.Body = ioutil.NopCloser(bytes.NewReader(body))

	var matched bool
	for _, cfg := range configs {
		for _, res := range cfg.Resources {
			if matchesResource(res, r, body) {
				matched = true
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
					fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n", r.Method, r.URL.Path, statusCode, len(data))
					return
				}

				if res.Response.Content != "" {
					fmt.Fprint(w, res.Response.Content)
					fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n", r.Method, r.URL.Path, statusCode, len(res.Response.Content))
					return
				}
			}
		}
	}

	if !matched {
		notFoundMsg := "Resource not found"
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, notFoundMsg)
		fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n", r.Method, r.URL.Path, http.StatusNotFound, len(notFoundMsg))
	}
}

// matchesResource checks if an incoming HTTP request matches the defined resource
func matchesResource(res config.Resource, r *http.Request, body []byte) bool {
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
