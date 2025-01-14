package xsd

import (
	"fmt"
	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/logger"
	"os"
	"strings"
)

// SplitQName splits a qualified name into namespace and local part
func SplitQName(qname string) (namespace, localPart string) {
	parts := strings.Split(qname, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", qname
}

// InheritNamespaces adds namespace prefixes to the schema if they are missing
// traversing up the parent nodes until the root
func InheritNamespaces(node *xmlquery.Node) {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		for _, attr := range parent.Attr {
			if doesSchemaHaveNsPrefix(node, attr.Name.Local) {
				continue
			}
			node.Attr = append(node.Attr, attr)
		}
	}
}

// doesSchemaHaveNsPrefix checks if the schema has a namespace prefix
func doesSchemaHaveNsPrefix(node *xmlquery.Node, prefix string) bool {
	for _, attr := range node.Attr {
		if attr.Name.Local == prefix {
			return true
		}
	}
	return false
}

// loadXmlFile loads an XML file, parses it, and returns the root node
func loadXmlFile(filePath string) (*xmlquery.Node, error) {
	// TODO cache this

	logger.Tracef("loading XML file: %s", filePath)
	schemaFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open XML file: %w", err)
	}
	defer schemaFile.Close()

	schemaDoc, err := xmlquery.Parse(schemaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load XML: %w", err)
	}
	return schemaDoc, nil
}

// GetTargetNamespace gets the target namespace from the schema document
func GetTargetNamespace(schemaDoc *xmlquery.Node) string {
	if schemaRoot := schemaDoc.SelectElement("schema"); schemaRoot != nil {
		if ns := schemaRoot.SelectAttr("targetNamespace"); ns != "" {
			return ns
		}
	}
	// Try to get from root element as fallback
	if root := schemaDoc.SelectElement("*"); root != nil {
		if ns := root.SelectAttr("targetNamespace"); ns != "" {
			return ns
		}
	}
	return ""
}
