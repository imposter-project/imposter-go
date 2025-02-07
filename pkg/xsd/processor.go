package xsd

import (
	"bytes"
	"fmt"
	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/pkg/logger"
	"os"
	"path/filepath"
	"strings"
)

// processSchema writes a schema and its imports to the destination directory, and returns a map of schema URLs to their local paths
func processSchema(wsdlDir string, schema *xmlquery.Node, destDir string, index int, processedSchemas *map[string]Schema, isXmlBaseSchema bool) error {
	if !isXmlBaseSchema {
		InheritNamespaces(schema)
	}

	schemaXML := schema.OutputXML(true)

	if !isXmlBaseSchema {
		schemaDoc, err := xmlquery.Parse(strings.NewReader(schemaXML))
		if err != nil {
			return fmt.Errorf("failed to parse schema XML: %w", err)
		}

		// Process imports first
		if err := processImports(wsdlDir, schemaDoc, processedSchemas, destDir); err != nil {
			return err
		}
	}

	// Write the current schema to the destination directory
	filename := fmt.Sprintf("schema_%d.xsd", index)
	schemaPath := filepath.Join(destDir, filename)
	if err := os.WriteFile(schemaPath, []byte(schemaXML), 0644); err != nil {
		return fmt.Errorf("failed to write schema to file: %w", err)
	}

	appendSchema(schema, processedSchemas, schemaPath)

	logger.Tracef("wrote schema %d to %s", index, schemaPath)
	return nil
}

// processImports processes import elements in a schema recursively
func processImports(wsdlDir string, schemaDoc *xmlquery.Node, processedSchemas *map[string]Schema, tempDir string) error {
	imports := xmlquery.Find(schemaDoc, ".//*[local-name()='import']")
	for _, imp := range imports {
		var schemaLocation, namespace string
		for _, attr := range imp.Attr {
			if attr.Name.Local == "schemaLocation" {
				schemaLocation = attr.Value
			} else if attr.Name.Local == "namespace" {
				namespace = attr.Value
			}
		}
		logger.Tracef("found import with schemaLocation: %s, namespace: %s", schemaLocation, namespace)

		if schemaLocation != "" && !isProcessed(schemaLocation, processedSchemas) {
			// Try to resolve the schema location relative to the WSDL directory
			resolvedPath := schemaLocation
			if !filepath.IsAbs(schemaLocation) {
				resolvedPath = filepath.Join(wsdlDir, schemaLocation)
			}
			logger.Tracef("resolved schema location: %s", resolvedPath)

			// Read and process the imported schema
			importedContent, err := os.ReadFile(resolvedPath)
			if err != nil {
				return fmt.Errorf("failed to read imported schema %s: %v", resolvedPath, err)
			}

			if err := importSchema(wsdlDir, tempDir, schemaLocation, importedContent, processedSchemas, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// importSchema processes a schema recursively
func importSchema(wsdlDir string, destDir string, schemaLocation string, schemaContent []byte, processedSchemas *map[string]Schema, isXmlBaseSchema bool) error {
	// Copy the imported schema to the temp directory
	targetPath := filepath.Join(destDir, filepath.Base(schemaLocation))
	if err := os.WriteFile(targetPath, schemaContent, 0644); err != nil {
		return fmt.Errorf("failed to write schema %s: %v", targetPath, err)
	}

	schemaDoc, err := xmlquery.Parse(bytes.NewReader(schemaContent))
	if err != nil {
		return fmt.Errorf("failed to parse schema %s: %v", schemaLocation, err)
	}

	schemaRootNode := xmlquery.FindOne(schemaDoc, "//*[local-name()='schema']")
	if schemaRootNode != nil {
		appendSchema(schemaRootNode, processedSchemas, targetPath)

		// Process the imported schema recursively
		subIndex := len(*processedSchemas)
		if err := processSchema(wsdlDir, schemaRootNode, destDir, subIndex, processedSchemas, isXmlBaseSchema); err != nil {
			return fmt.Errorf("failed to process schema %s: %v", schemaLocation, err)
		}
	}
	return nil
}

// appendSchema appends a schema to the processed schemas map
func appendSchema(schema *xmlquery.Node, processedSchemas *map[string]Schema, schemaPath string) {
	// note: an empty targetNamespace is valid

	targetNs := GetTargetNamespace(schema)
	(*processedSchemas)[getSchemaKey(schema)] = Schema{
		FilePath:        schemaPath,
		TargetNamespace: targetNs,
	}
}

// getSchemaKey generates a unique key for a schema node
func getSchemaKey(schema *xmlquery.Node) string {
	targetNs := ""
	for _, attr := range schema.Attr {
		if attr.Name.Local == "targetNamespace" {
			targetNs = attr.Value
			break
		}
	}
	return targetNs + "_" + schema.OutputXML(false)
}

// isProcessed checks if a schema has already been processed
func isProcessed(schemaLocation string, processedSchemas *map[string]Schema) bool {
	for _, schema := range *processedSchemas {
		if strings.Contains(schema.FilePath, schemaLocation) {
			return true
		}
	}
	return false
}
