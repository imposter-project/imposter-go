package main

import (
	"testing"
)

func TestLoadSchema(t *testing.T) {
	schema, err := loadSchema()
	if err != nil {
		t.Errorf("SchemaFile invalid.")
	}
	if schema != nil {

	}

}
