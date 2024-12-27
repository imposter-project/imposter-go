package matcher

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/gatehill/imposter-go/internal/config"
	"k8s.io/client-go/util/jsonpath"
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
		return MatchCondition("", condition.MatchCondition)
	}

	return MatchCondition(result.InnerText(), condition.MatchCondition)
}

// MatchJSONPath matches JSON body content using JSONPath query
func MatchJSONPath(body []byte, condition config.BodyMatchCondition) bool {
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return false
	}

	jpath := jsonpath.New("jsonpath")
	if err := jpath.Parse(condition.JSONPath); err != nil {
		return false
	}

	results := new(bytes.Buffer)
	if err := jpath.Execute(results, jsonData); err != nil {
		return false
	}

	return MatchCondition(results.String(), condition.MatchCondition)
}

// MatchSimpleOrAdvancedCondition checks if a value matches a condition based on the operator
func MatchSimpleOrAdvancedCondition(actualValue string, condition interface{}) bool {
	switch cond := condition.(type) {
	case string:
		return actualValue == cond
	case config.MatchCondition:
		return MatchCondition(actualValue, cond)
	default:
		return false
	}
}

// MatchCondition checks if a value matches a condition based on the operator
func MatchCondition(actualValue string, condition config.MatchCondition) bool {
	switch condition.Operator {
	case "EqualTo", "":
		return actualValue == condition.Value
	case "NotEqualTo":
		return actualValue != condition.Value
	case "Exists":
		return actualValue != ""
	case "NotExists":
		return actualValue == ""
	case "Contains":
		return strings.Contains(actualValue, condition.Value)
	case "NotContains":
		return !strings.Contains(actualValue, condition.Value)
	case "Matches":
		matched, _ := regexp.MatchString(condition.Value, actualValue)
		return matched
	case "NotMatches":
		matched, _ := regexp.MatchString(condition.Value, actualValue)
		return !matched
	default:
		return false
	}
}
