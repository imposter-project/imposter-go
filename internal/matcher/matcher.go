package matcher

import (
	"bytes"
	"encoding/json"

	"github.com/PaesslerAG/jsonpath"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/imposter-project/imposter-go/internal/config"
)

// MatchXPath matches XML body content using XPath query
func MatchXPath(body []byte, condition config.BodyMatchCondition) bool {
	doc, err := xmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		return false
	}

	// Compile an XPath expression with namespace bindings.
	// The map keys are the prefixes (e.g. "ns1"), and the values are the namespace URIs.
	expr, err := xpath.CompileWithNS(
		condition.XPath,
		condition.XMLNamespaces,
	)
	if err != nil {
		panic(err)
	}

	// Select the node using the compiled expression.
	result := xmlquery.QuerySelector(doc, expr)
	if result == nil {
		return condition.Match("")
	}

	return condition.Match(result.InnerText())
}

// MatchJSONPath matches JSON body content using JSONPath query
func MatchJSONPath(body []byte, condition config.BodyMatchCondition) bool {
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return false
	}

	results, err := jsonpath.Get(condition.JSONPath, jsonData)
	if err != nil {
		return false
	}

	return condition.Match(results.(string))
}
