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
	basePath, err := os.Getwd()
	filename := "imposter-config-schema.json"
	fullpath := filepath.Join(basePath, filename)
	withfileprefix := "file://" + fullpath
	schemaLoader := gojsonschema.NewReferenceLoader(withfileprefix)
	fmt.Println("Validating config files")
	filepath.Walk(*c, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}
		if info.IsDir() == false {
			if strings.Contains(path, "-config.yaml") {
				docYaml, err := os.ReadFile(path)
				j2, err := yaml.YAMLToJSON(docYaml)

				documentLoader := gojsonschema.NewStringLoader(string(j2))
				result, err := gojsonschema.Validate(schemaLoader, documentLoader)
				if err != nil {
					panic(err.Error())
				}

				if result.Valid() {
					fmt.Printf("✓ %s - Valid\n", path)
				} else {
					fmt.Printf("✗ %s - Invalid:\n", path)
					for _, desc := range result.Errors() {
						fmt.Printf("- %s\n", desc)
					}
				}
			}
		}
		return nil
	})
}
