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

func validateConfig(configDir string) int {
	fmt.Println("Validating config files")
	schemaLoader, err := loadSchema()
	if err != nil {
		panic(err.Error())
	}
	var configFiles = loadConfigFiles(configDir)
	var validFiles int
	for _, configFile := range configFiles {
		docYaml, err := os.ReadFile(configFile)
		if err != nil {
			panic(err.Error())
		}
		j2, err := yaml.YAMLToJSON(docYaml)
		if err != nil {
			panic(err.Error())
		}

		documentLoader := gojsonschema.NewStringLoader(string(j2))
		result, err := gojsonschema.Validate(schemaLoader, documentLoader)

		if err != nil {
			panic(err.Error())

		}

		if result.Valid() {
			fmt.Printf("✓ %s - Valid\n", configFile)
			validFiles++
		} else {
			fmt.Printf("✗ %s - Invalid:\n", configFile)
			for _, desc := range result.Errors() {
				fmt.Printf("\t - %s\n", desc)
			}
		}
	}
	return validFiles
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

	valid := validateConfig(*c)
	fmt.Printf("Successfully validated %d files.\n", valid)

}
