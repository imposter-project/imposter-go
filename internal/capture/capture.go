package capture

import (
	"fmt"
	"github.com/imposter-project/imposter-go/internal/query"
	"net/http"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/internal/template"
	"github.com/imposter-project/imposter-go/pkg/utils"
)

// CaptureRequestData captures elements of the request and stores them in the specified store.
func CaptureRequestData(imposterConfig *config.ImposterConfig, captureMap map[string]config.Capture, r *http.Request, body []byte, requestStore store.Store) {
	for key, capture := range captureMap {
		if capture.Enabled != nil && !*capture.Enabled {
			continue
		}

		itemName := getValueFromCaptureKey(capture.Key, key, r, body, imposterConfig, requestStore)
		value := getValueFromCaptureKey(capture.CaptureConfig, "", r, body, imposterConfig, requestStore)

		if value != "" {
			if capture.Store == "request" {
				requestStore[itemName] = value
			} else {
				store.StoreValue(capture.Store, itemName, value)
			}
		}
	}
}

// getValueFromCaptureKey retrieves the value based on the capture key configuration.
func getValueFromCaptureKey(key config.CaptureConfig, defaultKey string, r *http.Request, body []byte, imposterConfig *config.ImposterConfig, requestStore store.Store) string {
	if key.PathParam != "" {
		return utils.ExtractPathParams(r.URL.Path, r.URL.Path)[key.PathParam]
	} else if key.QueryParam != "" {
		return r.URL.Query().Get(key.QueryParam)
	} else if key.FormParam != "" {
		if err := r.ParseForm(); err == nil {
			return r.FormValue(key.FormParam)
		}
	} else if key.RequestHeader != "" {
		return r.Header.Get(key.RequestHeader)
	} else if key.Expression != "" {
		return template.ProcessTemplate(key.Expression, r, imposterConfig, requestStore)
	} else if key.Const != "" {
		return key.Const
	} else if key.RequestBody.JSONPath != "" {
		return extractJSONPath(body, key.RequestBody.JSONPath)
	} else if key.RequestBody.XPath != "" {
		return extractXPath(body, key.RequestBody.XPath, key.RequestBody.XMLNamespaces)
	}
	return defaultKey
}

// extractJSONPath extracts a value from the JSON body using a JSONPath expression.
func extractJSONPath(body []byte, jsonPathExpr string) string {
	result, success := query.JsonPathQuery(body, jsonPathExpr)
	if !success {
		return ""
	}
	return fmt.Sprintf("%v", result)
}

// extractXPath extracts a value from the XML body using an XPath expression.
func extractXPath(body []byte, xPath string, namespaces map[string]string) string {
	result, success := query.XPathQuery(body, xPath, namespaces)
	if !success {
		return ""
	}
	return result
}
