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

func CaptureRequestData(imposterConfig *config.ImposterConfig, resource config.Resource, r *http.Request, body []byte) {
	for key, capture := range resource.Capture {
		var value string
		if capture.PathParam != "" {
			value = utils.ExtractPathParams(r.URL.Path, resource.Path)[capture.PathParam]
		} else if capture.QueryParam != "" {
			value = r.URL.Query().Get(capture.QueryParam)
		} else if capture.FormParam != "" {
			if err := r.ParseForm(); err == nil {
				value = r.FormValue(capture.FormParam)
			}
		} else if capture.RequestHeader != "" {
			value = r.Header.Get(capture.RequestHeader)
		} else if capture.Expression != "" {
			value = template.ProcessTemplate(capture.Expression, r, imposterConfig)
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
