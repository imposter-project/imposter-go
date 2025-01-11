package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/akamensky/argparse"
	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"
)

func loadConfigFiles(configDir string) []string {
	var configFiles []string
	filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}
		if !info.IsDir() {
			if strings.Contains(path, "-config.y") {
				configFiles = append(configFiles, path)
			}
		}
		return nil
	})
	return configFiles
}

func validateConfig(schema gojsonschema.JSONLoader, configFile string) (int, error) {
	documentLoader := gojsonschema.NewStringLoader(configFile)
	result, err := gojsonschema.Validate(schema, documentLoader)

	if err != nil {
		panic(err.Error())
	}

	if result.Valid() {
		return 1, nil
	} else {
		for _, desc := range result.Errors() {
			fmt.Printf("\t - %s\n", desc)
		}
		return 0, nil
	}
}

func loadConfigFromFile(configDir string) (int, int, error) {
	var configFiles = loadConfigFiles(configDir)
	var validFiles int
	var totalFiles = len(configFiles)
	schema, err := loadSchema()
	if err != nil {
		panic(err.Error())
	}
	for _, configFile := range configFiles {
		docYaml, err := os.ReadFile(configFile)
		if err != nil {
			panic(err.Error())
		}
		j2, err := yaml.YAMLToJSON(docYaml)
		if err != nil {
			panic(err.Error())
		}
		fileResult, err := validateConfig(schema, string(j2))
		if err != nil {
			panic(err.Error())
		}
		if fileResult == 1 {
			fmt.Printf("✓ %s - Valid\n", configFile)
			validFiles = validFiles + fileResult
		} else {
			fmt.Printf("✗ %s - Invalid:\n", configFile)
		}
	}
	return validFiles, totalFiles, nil
}

func loadSchema() (gojsonschema.JSONLoader, error) {
	basePath, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	filename := "imposter-config-schema.json"
	fullpath := filepath.Join(basePath, filename)
	withfileprefix := "file://" + fullpath
	schemaLoader := gojsonschema.NewReferenceLoader(withfileprefix)
	return schemaLoader, nil
}

func main() {
	// Create a new parser
	parser := argparse.NewParser("validate_configs", "Validates your imposter configs against the imposter config schema.")
	// Create a ref to folder
	c := parser.String("c", "configs", &argparse.Options{Required: true, Help: "Location of config files"})

	err := parser.Parse(os.Args)

	if err != nil {
		// In case of error print error and print usage
		// This can also be done by passing -h or --help flags
		fmt.Print(parser.Usage(err))
	}
	if err != nil {
		log.Println(err)
	}

	valid, total, err := loadConfigFromFile(*c)
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("Successfully validated %d files of %d.\n", valid, total)

}
