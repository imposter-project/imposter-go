package soap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/imposter-project/imposter-go/internal/logger"
	"github.com/outofcoffee/go-xml-example-generator/examplegen"
)

// generateExampleXML generates example XML based on the WSDL schema
func generateExampleXML(element string, wsdlPath string, doc *xmlquery.Node) (string, error) {
	// Create a temporary directory for schema files
	tempDir, err := os.MkdirTemp("", "wsdl-schemas-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Process all schemas and their imports
	typesNode := xmlquery.FindOne(doc, "//*[local-name()='types']")
	if typesNode == nil {
		return "", fmt.Errorf("types element not found")
	}

	schemas := xmlquery.Find(typesNode, ".//*[local-name()='schema']")
	if len(schemas) == 0 {
		return "", fmt.Errorf("no schemas found")
	}

	// Track processed schemas to avoid duplicates
	processedSchemas := make(map[string]string)

	// Process each schema and its imports recursively
	wsdlDir := filepath.Dir(wsdlPath)
	for i, schema := range schemas {
		if err := processSchema(wsdlDir, schema, tempDir, i, processedSchemas); err != nil {
			return "", fmt.Errorf("failed to process schema %d: %w", i, err)
		}
	}

	// Get target namespace from first schema (or root if not found)
	var targetNS string
	for _, schemaNode := range schemas {
		if ns := schemaNode.SelectAttr("targetNamespace"); ns != "" {
			targetNS = ns
			break
		}
	}
	if targetNS == "" {
		// Try to get from root element as fallback
		if root := doc.SelectElement("*"); root != nil {
			targetNS = root.SelectAttr("targetNamespace")
		}
	}

	// Extract local part of element QName
	localPart := getLocalPart(element)
	prefix := getPrefix(element)

	elementExpr := fmt.Sprintf("//*[local-name()='element' and @name='%s']", localPart)
	
	// the path to the schema file that contains the element
	var elementSchemaPath string

	// If not found in main WSDL, try each processed schema
	if elementSchemaPath == "" {
		for _, schemaPath := range processedSchemas {
			schemaContent, err := os.ReadFile(schemaPath)
			if err != nil {
				logger.Warnf("failed to read schema file %s: %v", schemaPath, err)
				continue
			}

			schemaDoc, err := xmlquery.Parse(strings.NewReader(string(schemaContent)))
			if err != nil {
				logger.Warnf("failed to parse schema file %s: %v", schemaPath, err)
				continue
			}

			elementNode := xmlquery.FindOne(schemaDoc, elementExpr)
			if elementNode != nil {
				// Found the element, remember the path to the schema file and get its target namespace
				elementSchemaPath = schemaPath
				
				if schemaRoot := schemaDoc.SelectElement("schema"); schemaRoot != nil {
					if ns := schemaRoot.SelectAttr("targetNamespace"); ns != "" {
						targetNS = ns
					}
				}
				break
			}
		}
	}

	if elementSchemaPath == "" {
		return "", fmt.Errorf("element definition not found: %s (searched in WSDL and %d schema files)", element, len(processedSchemas))
	}

	logger.Debugf("generating example for element [localPart: %s, prefix: %s, target namespace: %s]", localPart, prefix, targetNS)

	example, err := examplegen.GenerateWithNs(elementSchemaPath, localPart, targetNS, prefix)
	if err != nil {
		return "", fmt.Errorf("failed to generate XML: %w", err)
	}

	return example, nil
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

			// Copy the imported schema to the temp directory
			targetPath := filepath.Join(tempDir, filepath.Base(schemaLocation))
			if err := os.WriteFile(targetPath, importedContent, 0644); err != nil {
				logger.Warnf("failed to write imported schema %s: %v", targetPath, err)
				continue
			}
			processedSchemas[schemaLocation] = targetPath

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
