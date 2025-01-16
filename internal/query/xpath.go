package query

import (
	"bytes"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/imposter-project/imposter-go/internal/logger"
)

// XPathQuery extracts a value from the XML document using an XPath expression.
func XPathQuery(body []byte, xPath string, namespaces map[string]string) (result string, success bool) {
	doc, err := xmlquery.Parse(bytes.NewReader(body))
	if err != nil {
		logger.Warnf("failed to parse XML data: %v", err)
		return "", false
	}

	if namespaces == nil {
		namespaces = make(map[string]string)
	}
	expr, err := xpath.CompileWithNS(xPath, namespaces)
	if err != nil {
		logger.Warnf("failed to compile XPath expression: %v", err)
		return "", false
	}

	queryResult := xmlquery.QuerySelector(doc, expr)
	if queryResult == nil {
		// empty is a valid result
		return "", true
	}

	return queryResult.InnerText(), true
}
