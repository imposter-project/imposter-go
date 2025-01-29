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
	// GetSchemas returns a map of schema URLs to schema metadata
	GetSchemas() map[string]Schema

	// GetSchemasWithTargetNamespace returns a slice of schemas with the given target namespace
	GetSchemasWithTargetNamespace(targetNamespace string) []Schema

	// ResolveElement resolves an element by QName
	ResolveElement(qname string) (*xml.Name, error)

	// ResolveType resolves a type by QName
	ResolveType(qname string) (*xml.Name, error)

	// ImportSchema imports a schema into the schema system
	ImportSchema(wsdlDir string, uniqueSchemaFilename string, schemaContent []byte) error
}

// Schema represents an XSD schema
type Schema struct {
	TargetNamespace string

	// FilePath is the local path of the schema
	FilePath string
}

type schemaSystem struct {
	// dir is the directory where schema files are stored
	dir     string
	schemas map[string]Schema
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
	processedSchemas := make(map[string]Schema)

	wsdlDir := filepath.Dir(wsdlPath)

	// Add the base XSD datatypes
	if err := importSchema(wsdlDir, tempDir, "XMLSchema.xsd", BaseDatatypes, &processedSchemas, true); err != nil {
		return nil, fmt.Errorf("failed to import base datatypes: %w", err)
	}

	// Process each schema (and its imports) recursively
	for i, schema := range schemas {
		if err := processSchema(wsdlDir, schema, tempDir, i, &processedSchemas, false); err != nil {
			return nil, fmt.Errorf("failed to process schema %d: %w", i, err)
		}
	}

	// Create a new schema system
	ss := &schemaSystem{
		dir:     tempDir,
		schemas: processedSchemas,
	}
	return ss, nil
}

// ImportSchema imports a schema into the schema system
func (s *schemaSystem) ImportSchema(wsdlDir string, uniqueSchemaFilename string, schemaContent []byte) error {
	logger.Tracef("importing schema: %s", uniqueSchemaFilename)
	if err := importSchema(wsdlDir, s.dir, uniqueSchemaFilename, schemaContent, &s.schemas, false); err != nil {
		return fmt.Errorf("failed to import schema: %s: %w", uniqueSchemaFilename, err)
	}
	return nil
}

func (s *schemaSystem) GetSchemas() map[string]Schema {
	return s.schemas
}

func (s *schemaSystem) GetSchemasWithTargetNamespace(targetNamespace string) []Schema {
	var schemas []Schema
	for _, schema := range s.GetSchemas() {
		if schema.TargetNamespace == targetNamespace {
			schemas = append(schemas, schema)
		}
	}
	return schemas
}

func (s *schemaSystem) ResolveElement(qname string) (*xml.Name, error) {
	_, localName := SplitQName(qname)
	for _, schema := range s.schemas {
		schemaDoc, err := loadXmlFile(schema.FilePath)
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
	for _, schema := range s.schemas {
		schemaDoc, err := loadXmlFile(schema.FilePath)
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
