package capture

import (
	"fmt"

	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/query"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/internal/template"
	"github.com/imposter-project/imposter-go/pkg/utils"
)

// CaptureRequestData captures elements of the request and stores them in the specified store.
func CaptureRequestData(imposterConfig *config.ImposterConfig, captureMap map[string]config.Capture, exch *exchange.Exchange) {
	for key, capture := range captureMap {
		if capture.Enabled != nil && !*capture.Enabled {
			continue
		}

		itemName := getValueFromCaptureKey(capture.Key, key, exch, imposterConfig)
		value := getValueFromCaptureKey(capture.CaptureConfig, "", exch, imposterConfig)

		if value != "" {
			s := store.Open(capture.Store, exch.RequestStore)
			s.StoreValue(itemName, value)
		}
	}
}

// getValueFromCaptureKey retrieves the value based on the capture key configuration.
func getValueFromCaptureKey(key config.CaptureConfig, defaultKey string, exch *exchange.Exchange, imposterConfig *config.ImposterConfig) string {
	if key.PathParam != "" {
		return utils.ExtractPathParams(exch.Request.Request.URL.Path, exch.Request.Request.URL.Path)[key.PathParam]
	} else if key.QueryParam != "" {
		return exch.Request.Request.URL.Query().Get(key.QueryParam)
	} else if key.FormParam != "" {
		if err := exch.Request.Request.ParseForm(); err == nil {
			return exch.Request.Request.FormValue(key.FormParam)
		}
	} else if key.RequestHeader != "" {
		return exch.Request.Request.Header.Get(key.RequestHeader)
	} else if key.Expression != "" {
		return template.ProcessTemplateWithContext(key.Expression, exch, imposterConfig)
	} else if key.Const != "" {
		return key.Const
	} else if key.RequestBody.JSONPath != "" {
		return extractJSONPath(exch.Request.Body, key.RequestBody.JSONPath)
	} else if key.RequestBody.XPath != "" {
		return extractXPath(exch.Request.Body, key.RequestBody.XPath, key.RequestBody.XMLNamespaces)
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
