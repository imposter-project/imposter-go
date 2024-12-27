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

	// Write response using 'best.Resource'
	statusCode := best.Resource.Response.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}
	w.WriteHeader(statusCode)

	if best.Resource.Response.File != "" {
		filePath := filepath.Join(configDir, best.Resource.Response.File)
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
		w.Write(data)
		fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
			r.Method, r.URL.Path, statusCode, len(data))
		return
	}

	if best.Resource.Response.Content != "" {
		fmt.Fprint(w, best.Resource.Response.Content)
		fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:%d\n",
			r.Method, r.URL.Path, statusCode, len(best.Resource.Response.Content))
		return
	}

	// If there's no file, no content, but we have a match:
	fmt.Printf("Handled request - method:%s, path:%s, status:%d, length:0\n",
		r.Method, r.URL.Path, statusCode)
}

// calculateMatchScore returns the number of matched constraints.
// Returns 0 if any required condition fails, meaning no match.
func calculateMatchScore(res config.Resource, r *http.Request, body []byte) int {
	score := 0

	// Match method
	if r.Method != res.Method {
		return 0
	}
	score++

	// Match path
	if r.URL.Path != res.Path {
		return 0
	}
	score++

	// Match query parameters
	for key, expectedValue := range res.QueryParams {
		actualValue := r.URL.Query().Get(key)
		if actualValue != expectedValue {
			return 0
		}
		score++
	}

	// Match headers
	for key, expectedValue := range res.Headers {
		actualValue := r.Header.Get(key)
		if !strings.EqualFold(actualValue, expectedValue) {
			return 0
		}
		score++
	}

	// Match form parameters (if content type is application/x-www-form-urlencoded)
	if len(res.FormParams) > 0 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			return 0
		}
		for key, expectedValue := range res.FormParams {
			if r.FormValue(key) != expectedValue {
				return 0
			}
			score++
		}
	}

	// Match request body
	if xpathQuery, exists := res.RequestBody["xpath"]; exists {
		if !matcher.MatchXPath(body, xpathQuery) {
			return 0
		}
		score++
	}
	if jsonPathQuery, exists := res.RequestBody["jsonpath"]; exists {
		if !matcher.MatchJSONPath(body, jsonPathQuery) {
			return 0
		}
		score++
	}

	return score
}
