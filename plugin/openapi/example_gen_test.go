package openapi

import (
	"encoding/json"
	uuid "github.com/satori/go.uuid"
	"testing"
	"time"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerateExampleJSON(t *testing.T) {
	tests := []struct {
		name     string
		response Response
		want     string
		wantErr  bool
	}{
		{
			name: "with example",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Examples: map[string]string{
						defaultExampleName: `{"name": "test"}`,
					},
				},
			},
			want: `{"name": "test"}`,
		},
		{
			name: "with string schema",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Schema: createSchemaProxy([]string{"string"}, "", nil, nil, nil),
				},
			},
			want: `"example"`,
		},
		{
			name: "with number schema",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Schema: createSchemaProxy([]string{"number"}, "", nil, nil, nil),
				},
			},
			want: "42",
		},
		{
			name: "with boolean schema",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Schema: createSchemaProxy([]string{"boolean"}, "", nil, nil, nil),
				},
			},
			want: "false",
		},
		{
			name: "with array schema",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Schema: createSchemaProxy([]string{"array"}, "", nil, createSchemaProxy([]string{"string"}, "", nil, nil, nil), nil),
				},
			},
			want: `["example"]`,
		},
		{
			name: "with object schema",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Schema: createSchemaProxy([]string{"object"}, "", map[string]*base.SchemaProxy{
						"name": createSchemaProxy([]string{"string"}, "", nil, nil, nil),
						"age":  createSchemaProxy([]string{"integer"}, "", nil, nil, nil),
					}, nil, nil),
				},
			},
			want: `{"age":42,"name":"example"}`,
		},
		{
			name: "with enum",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Schema: createSchemaProxyWithEnum([]string{"string"}, "", []interface{}{"one", "two", "three"}),
				},
			},
			want: `"one"`,
		},
		{
			name: "with date-time format",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Schema: createSchemaProxy([]string{"string"}, "date-time", nil, nil, nil),
				},
			},
			want: `"` + time.Now().UTC().Format(time.RFC3339) + `"`,
		},
		{
			name: "with allOf",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Schema: createSchemaProxyWithAllOf([]*base.SchemaProxy{
						createSchemaProxy([]string{"object"}, "", map[string]*base.SchemaProxy{
							"name": createSchemaProxy([]string{"string"}, "", nil, nil, nil),
						}, nil, nil),
						createSchemaProxy([]string{"object"}, "", map[string]*base.SchemaProxy{
							"age": createSchemaProxy([]string{"integer"}, "", nil, nil, nil),
						}, nil, nil),
					}),
				},
			},
			want: `{"age":42,"name":"example"}`,
		},
		{
			name: "with oneOf",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Schema: createSchemaProxyWithOneOf([]*base.SchemaProxy{
						createSchemaProxy([]string{"string"}, "", nil, nil, nil),
						createSchemaProxy([]string{"integer"}, "", nil, nil, nil),
					}),
				},
			},
			want: `"example"`,
		},
		{
			name: "with anyOf",
			response: Response{
				UniqueID: uuid.NewV4().String(),
				SparseResponse: SparseResponse{
					Schema: createSchemaProxyWithAnyOf([]*base.SchemaProxy{
						createSchemaProxy([]string{"string"}, "", nil, nil, nil),
						createSchemaProxy([]string{"integer"}, "", nil, nil, nil),
					}),
				},
			},
			want: `"example"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateExampleJSON(tt.response.SparseResponse, defaultExampleName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// For date-time format, we need to parse and compare the time
			if tt.response.Schema != nil && tt.response.Schema.Schema().Format == "date-time" {
				gotTime, err := time.Parse(time.RFC3339, got[1:len(got)-1]) // Remove quotes
				require.NoError(t, err)
				wantTime, err := time.Parse(time.RFC3339, tt.want[1:len(tt.want)-1]) // Remove quotes
				require.NoError(t, err)
				assert.True(t, gotTime.Equal(wantTime))
				return
			}

			// For other cases, we can compare the JSON directly
			var gotJSON, wantJSON interface{}
			err = json.Unmarshal([]byte(got), &gotJSON)
			require.NoError(t, err)
			err = json.Unmarshal([]byte(tt.want), &wantJSON)
			require.NoError(t, err)
			assert.Equal(t, wantJSON, gotJSON)
		})
	}
}

// Helper function to create a SchemaProxy for testing
func createSchemaProxy(schemaType []string, format string, properties map[string]*base.SchemaProxy, items *base.SchemaProxy, enum []interface{}) *base.SchemaProxy {
	schema := &base.Schema{
		Type:   schemaType,
		Format: format,
	}

	if properties != nil {
		schema.Properties = orderedmap.New[string, *base.SchemaProxy]()
		for k, v := range properties {
			schema.Properties.Set(k, v)
		}
	}

	if items != nil {
		schema.Items = &base.DynamicValue[*base.SchemaProxy, bool]{
			A: items,
		}
	}

	return base.CreateSchemaProxy(schema)
}

// Helper function to create a SchemaProxy with enum values for testing
func createSchemaProxyWithEnum(schemaType []string, format string, enumValues []interface{}) *base.SchemaProxy {
	schema := &base.Schema{
		Type:   schemaType,
		Format: format,
		Enum:   make([]*yaml.Node, len(enumValues)),
	}

	// Convert enum values to yaml.Node
	for i, v := range enumValues {
		node := &yaml.Node{}
		if err := node.Encode(v); err == nil {
			schema.Enum[i] = node
		}
	}

	return base.CreateSchemaProxy(schema)
}

// Helper function to create a SchemaProxy with allOf for testing
func createSchemaProxyWithAllOf(schemas []*base.SchemaProxy) *base.SchemaProxy {
	schema := &base.Schema{
		AllOf: schemas,
	}
	return base.CreateSchemaProxy(schema)
}

// Helper function to create a SchemaProxy with oneOf for testing
func createSchemaProxyWithOneOf(schemas []*base.SchemaProxy) *base.SchemaProxy {
	schema := &base.Schema{
		OneOf: schemas,
	}
	return base.CreateSchemaProxy(schema)
}

// Helper function to create a SchemaProxy with anyOf for testing
func createSchemaProxyWithAnyOf(schemas []*base.SchemaProxy) *base.SchemaProxy {
	schema := &base.Schema{
		AnyOf: schemas,
	}
	return base.CreateSchemaProxy(schema)
}
