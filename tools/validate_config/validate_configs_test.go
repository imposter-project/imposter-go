package main

import (
	"fmt"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	c := "../../examples"
	got := validateConfig(c)
	if got != 7 {
		t.Errorf("Expected 7 but got %d", got)
	}
}

func TestLoadSchema(t *testing.T) {
	schema, err := loadSchema()
	if err != nil {
		t.Errorf("SchemaFile invalid.")
	}
	fmt.Println(schema)

}
