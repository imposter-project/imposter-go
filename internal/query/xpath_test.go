package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXPathQuery(t *testing.T) {
	tests := []struct {
		name          string
		xml           []byte
		xpathExpr     string
		namespaces    map[string]string
		expectedValue string
		expectSuccess bool
	}{
		{
			name:          "simple element value",
			xml:           []byte(`<root><name>test</name></root>`),
			xpathExpr:     "/root/name",
			namespaces:    nil,
			expectedValue: "test",
			expectSuccess: true,
		},
		{
			name:          "nested element value",
			xml:           []byte(`<person><details><age>30</age></details></person>`),
			xpathExpr:     "/person/details/age",
			namespaces:    nil,
			expectedValue: "30",
			expectSuccess: true,
		},
		{
			name:          "attribute value",
			xml:           []byte(`<user id="123"><name>John</name></user>`),
			xpathExpr:     "/user/@id",
			namespaces:    nil,
			expectedValue: "123",
			expectSuccess: true,
		},
		{
			name:          "with namespace",
			xml:           []byte(`<ns1:root xmlns:ns1="http://example.com"><ns1:name>test</ns1:name></ns1:root>`),
			xpathExpr:     "/ns:root/ns:name",
			namespaces:    map[string]string{"ns": "http://example.com"},
			expectedValue: "test",
			expectSuccess: true,
		},
		{
			name:          "invalid XML",
			xml:           []byte(`<invalid>`),
			xpathExpr:     "/root/name",
			namespaces:    nil,
			expectedValue: "",
			expectSuccess: false,
		},
		{
			name:          "invalid XPath expression",
			xml:           []byte(`<root><name>test</name></root>`),
			xpathExpr:     "///invalid",
			namespaces:    nil,
			expectedValue: "",
			expectSuccess: false,
		},
		{
			name:          "non-existent path",
			xml:           []byte(`<root><name>test</name></root>`),
			xpathExpr:     "/root/age",
			namespaces:    nil,
			expectedValue: "",
			expectSuccess: true,
		},
		{
			name: "complex nested structure",
			xml: []byte(`
				<users>
					<user>
						<name>John</name>
						<scores>
							<score>85</score>
							<score>90</score>
						</scores>
					</user>
					<user>
						<name>Jane</name>
						<scores>
							<score>95</score>
							<score>98</score>
						</scores>
					</user>
				</users>`),
			xpathExpr:     "/users/user[2]/scores/score[2]",
			namespaces:    nil,
			expectedValue: "98",
			expectSuccess: true,
		},
		{
			name:          "empty element",
			xml:           []byte(`<root><empty></empty></root>`),
			xpathExpr:     "/root/empty",
			namespaces:    nil,
			expectedValue: "",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, success := XPathQuery(tt.xml, tt.xpathExpr, tt.namespaces)
			assert.Equal(t, tt.expectSuccess, success)
			if tt.expectSuccess {
				assert.Equal(t, tt.expectedValue, result)
			}
		})
	}
}
