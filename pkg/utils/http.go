package utils

import "strings"

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
