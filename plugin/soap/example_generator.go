package soap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/outofcoffee/go-xml-example-generator/examplegen"
)

// generateExampleXML generates example XML based on the WSDL schema
func generateExampleXML(elementName string, wsdlPath string, wsdlDoc *xmlquery.Node) (string, error) {
	wsdlDir := filepath.Dir(wsdlPath)

	// extract all schemas from the wsdlDoc
	typesNode := xmlquery.FindOne(wsdlDoc, "//*[local-name()='types']")
	if typesNode == nil {
		return "", errors.New("types element not found")
	}

	schemas := xmlquery.Find(typesNode, ".//*[local-name()='schema']")
	if len(schemas) == 0 {
		return "", errors.New("no schemas found")
	}

	// Create a temporary directory for all schema files
	tempDir, err := os.MkdirTemp("", "schema_files")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Track processed schemas to avoid duplicates
	processedSchemas := make(map[string]string)

	// Process each schema and its imports recursively
	for i, schema := range schemas {
		if err := processSchema(wsdlDir, schema, tempDir, i, processedSchemas); err != nil {
			return "", fmt.Errorf("failed to process schema %d: %w", i, err)
		}
	}

	// Create an umbrella schema that imports all the processed schemas
	umbrellaFile := filepath.Join(tempDir, "umbrella_schema.xsd")
	umbrellaSchema := `<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`

	for _, schemaPath := range processedSchemas {
		targetNs := extractTargetNamespace(schemaPath)
		relPath, _ := filepath.Rel(filepath.Dir(umbrellaFile), schemaPath)

		if targetNs != "" {
			umbrellaSchema += fmt.Sprintf(`
    <xs:import namespace="%s" schemaLocation="%s"/>`, targetNs, relPath)
		} else {
			umbrellaSchema += fmt.Sprintf(`
    <xs:include schemaLocation="%s"/>`, relPath)
		}
	}

	umbrellaSchema += `
</xs:schema>`

	if err := os.WriteFile(umbrellaFile, []byte(umbrellaSchema), 0644); err != nil {
		return "", fmt.Errorf("failed to write umbrella schema: %w", err)
	}
	logger.Tracef("wrote umbrella schema to %s", umbrellaFile)

	// Generate XML using the umbrella schema
	xml, err := examplegen.Generate(umbrellaFile, elementName)
	if err != nil {
		return "", fmt.Errorf("failed to generate XML: %w", err)
	}

	logger.Tracef("generated XML: %s", xml)
	return xml, nil
}

// processSchema writes a schema to a temporary file and processes its imports
func processSchema(wsdlDir string, schema *xmlquery.Node, tempDir string, index int, processedSchemas map[string]string) error {
	schemaXML := schema.OutputXML(true)
	schemaDoc, err := xmlquery.Parse(strings.NewReader(schemaXML))
	if err != nil {
		return fmt.Errorf("failed to parse schema XML: %w", err)
	}

	// Process imports first
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
				logger.Warnf("failed to read imported schema %s: %v", resolvedPath, err)
				continue
			}

			importedDoc, err := xmlquery.Parse(strings.NewReader(string(importedContent)))
			if err != nil {
				logger.Warnf("failed to parse imported schema %s: %v", resolvedPath, err)
				continue
			}

			importedSchema := xmlquery.FindOne(importedDoc, "//*[local-name()='schema']")
			if importedSchema != nil {
				// Process the imported schema recursively
				subIndex := len(processedSchemas)
				if err := processSchema(wsdlDir, importedSchema, tempDir, subIndex, processedSchemas); err != nil {
					logger.Warnf("failed to process imported schema %s: %v", resolvedPath, err)
				}
			}
		}
	}

	// Write the current schema to a temporary file
	filename := fmt.Sprintf("schema_%d.xsd", index)
	schemaPath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(schemaPath, []byte(schemaXML), 0644); err != nil {
		return fmt.Errorf("failed to write schema to file: %w", err)
	}

	processedSchemas[getSchemaKey(schema)] = schemaPath
	logger.Tracef("wrote schema %d to %s", index, schemaPath)
	return nil
}

// extractTargetNamespace gets the target namespace from a schema file
func extractTargetNamespace(schemaPath string) string {
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return ""
	}

	doc, err := xmlquery.Parse(strings.NewReader(string(content)))
	if err != nil {
		return ""
	}

	schemaNode := xmlquery.FindOne(doc, "//*[local-name()='schema']")
	if schemaNode == nil {
		return ""
	}

	for _, attr := range schemaNode.Attr {
		if attr.Name.Local == "targetNamespace" {
			return attr.Value
		}
	}
	return ""
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
func isProcessed(schemaLocation string, processedSchemas map[string]string) bool {
	for _, path := range processedSchemas {
		if strings.Contains(path, schemaLocation) {
			return true
		}
	}
	return false
}
