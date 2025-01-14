package xsd

import (
	"encoding/xml"
	"fmt"
	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/logger"
	"os"
	"path/filepath"
)

// SchemaSystem is an interface for resolving schema elements and types
type SchemaSystem interface {
	// GetSchemas returns a map of schema URLs to their local paths
	GetSchemas() map[string]string

	// ResolveElement resolves an element by QName
	ResolveElement(qname string) (*xml.Name, error)

	// ResolveType resolves a type by QName
	ResolveType(qname string) (*xml.Name, error)
}

type schemaSystem struct {
	schemas map[string]string
}

// ExtractSchemas extracts schemas from a WSDL document and returns a schema system
func ExtractSchemas(wsdlPath string, wsdlDoc *xmlquery.Node) (SchemaSystem, error) {
	// Create a temporary directory for schema files
	tempDir, err := os.MkdirTemp("", "wsdl-schemas-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	//defer os.RemoveAll(tempDir)

	var schemas []*xmlquery.Node

	// Process all schemas and their imports
	typesNode := xmlquery.FindOne(wsdlDoc, "//*[local-name()='types']")
	if typesNode == nil {
		logger.Warnf("types element not found")
	} else {
		schemas = xmlquery.Find(typesNode, ".//*[local-name()='schema']")
	}
	if len(schemas) == 0 {
		// only base XSD datatypes are supported
		logger.Warnf("no schemas found")
	}

	// Track processed schemas to avoid duplicates
	processedSchemas := make(map[string]string)

	wsdlDir := filepath.Dir(wsdlPath)

	// Add the base XSD datatypes
	if err := importSchema(wsdlDir, tempDir, "XMLSchema-datatypes.xsd", BaseDatatypes, processedSchemas, "XMLSchema-datatypes.xsd"); err != nil {
		return nil, fmt.Errorf("failed to import base datatypes: %w", err)
	}

	// Process each schema (and its imports) recursively
	for i, schema := range schemas {
		if err := processSchema(wsdlDir, schema, tempDir, i, processedSchemas); err != nil {
			return nil, fmt.Errorf("failed to process schema %d: %w", i, err)
		}
	}

	// Create a new schema system
	ss := &schemaSystem{
		schemas: processedSchemas,
	}
	return ss, nil
}

func (s *schemaSystem) GetSchemas() map[string]string {
	return s.schemas
}

func (s *schemaSystem) ResolveElement(qname string) (*xml.Name, error) {
	_, localName := SplitQName(qname)
	for _, schemaPath := range s.schemas {
		schemaDoc, err := loadXmlFile(schemaPath)
		if err != nil {
			return nil, err
		}

		// Find the element with the given local name
		element := xmlquery.FindOne(schemaDoc, fmt.Sprintf("//*[local-name()='element' and @name='%s']", localName))
		if element != nil {
			elName := &xml.Name{
				Space: GetTargetNamespace(schemaDoc),
				Local: element.SelectAttr("name"),
			}
			return elName, nil
		}
	}
	return nil, fmt.Errorf("element %s not found", qname)
}

func (s *schemaSystem) ResolveType(qname string) (*xml.Name, error) {
	_, localName := SplitQName(qname)
	for _, schemaPath := range s.schemas {
		schemaDoc, err := loadXmlFile(schemaPath)
		if err != nil {
			return nil, err
		}

		// Find the complexType with the given local name
		typ := xmlquery.FindOne(schemaDoc, fmt.Sprintf("//*[local-name()='complexType' and @name='%s']", localName))
		if typ != nil {
			typName := &xml.Name{
				Space: GetTargetNamespace(schemaDoc),
				Local: typ.SelectAttr("name"),
			}
			return typName, nil
		}

		// Find the simpleType with the given local name
		typ = xmlquery.FindOne(schemaDoc, fmt.Sprintf("//*[local-name()='simpleType' and @name='%s']", localName))
		if typ != nil {
			typName := &xml.Name{
				Space: GetTargetNamespace(schemaDoc),
				Local: typ.SelectAttr("name"),
			}
			return typName, nil
		}
	}
	return nil, fmt.Errorf("type %s not found", qname)
}
