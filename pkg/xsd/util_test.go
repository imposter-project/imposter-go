package xsd

import (
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/stretchr/testify/require"
)

func TestInheritNamespaces_ParentHasNamespaces_InheritsSuccessfully(t *testing.T) {
	xmlContent := `
	<root xmlns:ns1="http://example.com/ns1">
		<parent>
			<child></child>
		</parent>
	</root>`
	doc, err := xmlquery.Parse(strings.NewReader(xmlContent))
	require.NoError(t, err)

	childNode := xmlquery.FindOne(doc, "//child")
	require.NotNil(t, childNode)

	InheritNamespaces(childNode)

	nsAttr := childNode.SelectAttr("xmlns:ns1")
	require.Equal(t, "http://example.com/ns1", nsAttr)
}

func TestInheritNamespaces_NoParentNamespaces_NoInheritance(t *testing.T) {
	xmlContent := `
	<root>
		<parent>
			<child></child>
		</parent>
	</root>`
	doc, err := xmlquery.Parse(strings.NewReader(xmlContent))
	require.NoError(t, err)

	childNode := xmlquery.FindOne(doc, "//child")
	require.NotNil(t, childNode)

	InheritNamespaces(childNode)

	nsAttr := childNode.SelectAttr("xmlns:ns1")
	require.Equal(t, "", nsAttr)
}

func TestInheritNamespaces_MultipleParentNamespaces_InheritsAll(t *testing.T) {
	xmlContent := `
	<root xmlns:ns1="http://example.com/ns1">
		<parent xmlns:ns2="http://example.com/ns2">
			<child></child>
		</parent>
	</root>`
	doc, err := xmlquery.Parse(strings.NewReader(xmlContent))
	require.NoError(t, err)

	childNode := xmlquery.FindOne(doc, "//child")
	require.NotNil(t, childNode)

	InheritNamespaces(childNode)

	ns1Attr := childNode.SelectAttr("xmlns:ns1")
	require.Equal(t, "http://example.com/ns1", ns1Attr)

	ns2Attr := childNode.SelectAttr("xmlns:ns2")
	require.Equal(t, "http://example.com/ns2", ns2Attr)
}

func TestInheritNamespaces_ChildAlreadyHasNamespace_NoOverride(t *testing.T) {
	xmlContent := `
	<root xmlns:ns1="http://example.com/ns1">
		<parent>
			<child xmlns:ns1="http://example.com/child-ns1"></child>
		</parent>
	</root>`
	doc, err := xmlquery.Parse(strings.NewReader(xmlContent))
	require.NoError(t, err)

	childNode := xmlquery.FindOne(doc, "//child")
	require.NotNil(t, childNode)

	InheritNamespaces(childNode)

	nsAttr := childNode.SelectAttr("xmlns:ns1")
	require.Equal(t, "http://example.com/child-ns1", nsAttr)
}
