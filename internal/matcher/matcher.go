package matcher

import (
	"bytes"
	"encoding/json"
	"github.com/antchfx/xmlquery"
	"k8s.io/client-go/util/jsonpath"
)

// MatchXPath matches XML body content using XPath query
func MatchXPath(body []byte, query string) bool {
	doc, err := xmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		return false
	}

	return xmlquery.FindOne(doc, query) != nil
}

// MatchJSONPath matches JSON body content using JSONPath query
func MatchJSONPath(body []byte, query string) bool {
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return false
	}

	jpath := jsonpath.New("jsonpath")
	if err := jpath.Parse(query); err != nil {
		return false
	}

	results := new(bytes.Buffer)
	if err := jpath.Execute(results, jsonData); err != nil {
		return false
	}

	return results.Len() > 0
}