package utils

import (
	"net/http"
	"strings"
)

// defaultMaxMultipartMemory mirrors net/http's defaultMaxMemory used by
// ParseMultipartForm when called via FormValue (32 MiB).
const defaultMaxMultipartMemory = 32 << 20

// ExtractPathParams extracts path parameters from the request path
func ExtractPathParams(requestPath, resourcePath string) map[string]string {
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

// parseRequestForms ensures both URL-encoded and multipart form bodies are
// parsed. Go's ParseForm only handles application/x-www-form-urlencoded; once
// it has run, r.Form is non-nil and FormValue will no longer trigger
// ParseMultipartForm on its own, so we must call it explicitly for multipart
// requests.
func parseRequestForms(r *http.Request) {
	_ = r.ParseForm()
	if isMultipartFormRequest(r) {
		_ = r.ParseMultipartForm(defaultMaxMultipartMemory)
	}
}

func isMultipartFormRequest(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data")
}

// GetFormParams returns a flat map of form parameters from the request,
// handling both application/x-www-form-urlencoded and multipart/form-data
// bodies. Only text fields are included; file parts are ignored. For
// multi-valued parameters, only the first value is returned.
func GetFormParams(r *http.Request) map[string]string {
	formParams := make(map[string]string)
	parseRequestForms(r)

	for k, v := range r.PostForm {
		if len(v) > 0 {
			formParams[k] = v[0]
		}
	}
	if r.MultipartForm != nil {
		for k, v := range r.MultipartForm.Value {
			if _, ok := formParams[k]; ok {
				continue
			}
			if len(v) > 0 {
				formParams[k] = v[0]
			}
		}
	}
	return formParams
}

// GetFormValue returns the value of a single form parameter, handling both
// application/x-www-form-urlencoded and multipart/form-data bodies.
func GetFormValue(r *http.Request, key string) string {
	parseRequestForms(r)
	return r.FormValue(key)
}
