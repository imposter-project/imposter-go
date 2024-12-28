package capture

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/gatehill/imposter-go/internal/config"
	"github.com/gatehill/imposter-go/internal/store"
	"github.com/gatehill/imposter-go/internal/template"
	"github.com/gatehill/imposter-go/pkg/utils"
	"k8s.io/client-go/util/jsonpath"
)

// CaptureRequestData captures elements of the request and stores them in the specified store.
func CaptureRequestData(imposterConfig *config.ImposterConfig, resource config.Resource, r *http.Request, body []byte, requestStore map[string]interface{}) {
	for key, capture := range resource.Capture {
		if !capture.Enabled {
			continue
		}

		itemName := getValueFromCaptureKey(capture.Key, key, r, body, imposterConfig, requestStore)
		value := getValueFromCaptureKey(capture.CaptureKey, "", r, body, imposterConfig, requestStore)

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
func getValueFromCaptureKey(key config.CaptureKey, defaultKey string, r *http.Request, body []byte, imposterConfig *config.ImposterConfig, requestStore map[string]interface{}) string {
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

// extractXPath extracts a value from the XML body using an XPath expression.
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
