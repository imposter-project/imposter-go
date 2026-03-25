package openapi

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/imposter-project/imposter-go/internal/fakedata"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	uuid "github.com/satori/go.uuid"
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
				// Allow for small time differences due to test execution timing
				timeDiff := gotTime.Sub(wantTime)
				if timeDiff < 0 {
					timeDiff = -timeDiff
				}
				assert.True(t, timeDiff < 10*time.Second, "time difference %v exceeds 10 second tolerance", timeDiff)
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

// mockOAFakeDataProvider is a test double for fakedata.Provider in OpenAPI tests.
type mockOAFakeDataProvider struct{}

func (m *mockOAFakeDataProvider) GenerateFakeData(req fakedata.Request) fakedata.Response {
	// Expression-based
	if req.ExprCategory == "Color" && req.ExprProperty == "name" {
		return fakedata.Response{Value: "blue", Found: true}
	}
	if req.ExprCategory == "Name" && req.ExprProperty == "firstName" {
		return fakedata.Response{Value: "Jane", Found: true}
	}
	// Property name inference
	if req.PropertyName == "firstName" {
		return fakedata.Response{Value: "Jane", Found: true}
	}
	if req.PropertyName == "email" {
		return fakedata.Response{Value: "jane@example.com", Found: true}
	}
	if req.PropertyName == "city" {
		return fakedata.Response{Value: "Portland", Found: true}
	}
	// Format inference
	if req.Format == "email" {
		return fakedata.Response{Value: "fake@example.com", Found: true}
	}
	return fakedata.Response{}
}

// createSchemaProxyWithExtension creates a SchemaProxy with an x-fake-data extension.
func createSchemaProxyWithExtension(schemaType []string, fakeDataValue string) *base.SchemaProxy {
	schema := &base.Schema{
		Type: schemaType,
	}
	extensions := orderedmap.New[string, *yaml.Node]()
	extensions.Set("x-fake-data", &yaml.Node{Kind: yaml.ScalarNode, Value: fakeDataValue})
	schema.Extensions = extensions
	return base.CreateSchemaProxy(schema)
}

func TestGenerateExampleJSON_WithFakeDataExtension(t *testing.T) {
	fakedata.RegisterProvider(&mockOAFakeDataProvider{})
	defer fakedata.RegisterProvider(nil)

	response := Response{
		UniqueID: uuid.NewV4().String(),
		SparseResponse: SparseResponse{
			Schema: createSchemaProxyWithExtension([]string{"string"}, "Color.name"),
		},
	}

	got, err := generateExampleJSON(response.SparseResponse, defaultExampleName)
	require.NoError(t, err)
	assert.Equal(t, `"blue"`, got)
}

func TestGenerateExampleJSON_WithPropertyNameInference(t *testing.T) {
	fakedata.RegisterProvider(&mockOAFakeDataProvider{})
	defer fakedata.RegisterProvider(nil)

	// Create an object schema with properties that should be inferred
	response := Response{
		UniqueID: uuid.NewV4().String(),
		SparseResponse: SparseResponse{
			Schema: createSchemaProxy([]string{"object"}, "", map[string]*base.SchemaProxy{
				"firstName": createSchemaProxy([]string{"string"}, "", nil, nil, nil),
				"email":     createSchemaProxy([]string{"string"}, "", nil, nil, nil),
				"city":      createSchemaProxy([]string{"string"}, "", nil, nil, nil),
				"age":       createSchemaProxy([]string{"integer"}, "", nil, nil, nil),
			}, nil, nil),
		},
	}

	got, err := generateExampleJSON(response.SparseResponse, defaultExampleName)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(got), &result)
	require.NoError(t, err)

	// firstName, email, city should be inferred from property names
	assert.Equal(t, "Jane", result["firstName"])
	assert.Equal(t, "jane@example.com", result["email"])
	assert.Equal(t, "Portland", result["city"])
	// age should fall through to default number generation
	assert.Equal(t, float64(42), result["age"])
}

func TestGenerateExampleJSON_WithFormatFakeData(t *testing.T) {
	fakedata.RegisterProvider(&mockOAFakeDataProvider{})
	defer fakedata.RegisterProvider(nil)

	response := Response{
		UniqueID: uuid.NewV4().String(),
		SparseResponse: SparseResponse{
			Schema: createSchemaProxy([]string{"string"}, "email", nil, nil, nil),
		},
	}

	got, err := generateExampleJSON(response.SparseResponse, defaultExampleName)
	require.NoError(t, err)
	assert.Equal(t, `"fake@example.com"`, got)
}

func TestGenerateExampleJSON_WithFakeDataExtensionOnProperty(t *testing.T) {
	fakedata.RegisterProvider(&mockOAFakeDataProvider{})
	defer fakedata.RegisterProvider(nil)

	// Create an object with a property that has x-fake-data extension
	props := map[string]*base.SchemaProxy{
		"favoriteColor": createSchemaProxyWithExtension([]string{"string"}, "Color.name"),
	}
	schema := &base.Schema{
		Type: []string{"object"},
	}
	schema.Properties = orderedmap.New[string, *base.SchemaProxy]()
	for k, v := range props {
		schema.Properties.Set(k, v)
	}

	response := Response{
		UniqueID: uuid.NewV4().String(),
		SparseResponse: SparseResponse{
			Schema: base.CreateSchemaProxy(schema),
		},
	}

	got, err := generateExampleJSON(response.SparseResponse, defaultExampleName)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(got), &result)
	require.NoError(t, err)

	assert.Equal(t, "blue", result["favoriteColor"])
}

func TestGenerateExampleJSON_NoProviderFallback(t *testing.T) {
	// Ensure no provider is registered
	fakedata.RegisterProvider(nil)

	// With no provider, should fall back to default behavior
	response := Response{
		UniqueID: uuid.NewV4().String(),
		SparseResponse: SparseResponse{
			Schema: createSchemaProxy([]string{"object"}, "", map[string]*base.SchemaProxy{
				"firstName": createSchemaProxy([]string{"string"}, "", nil, nil, nil),
			}, nil, nil),
		},
	}

	got, err := generateExampleJSON(response.SparseResponse, defaultExampleName)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(got), &result)
	require.NoError(t, err)

	// Without provider, should fall back to "example"
	assert.Equal(t, "example", result["firstName"])
}

func TestGetFakeDataExtension(t *testing.T) {
	// With extension
	schema := &base.Schema{Type: []string{"string"}}
	extensions := orderedmap.New[string, *yaml.Node]()
	extensions.Set("x-fake-data", &yaml.Node{Kind: yaml.ScalarNode, Value: "Name.firstName"})
	schema.Extensions = extensions

	assert.Equal(t, "Name.firstName", getFakeDataExtension(schema))

	// Without extension
	schema2 := &base.Schema{Type: []string{"string"}}
	assert.Equal(t, "", getFakeDataExtension(schema2))

	// With nil extensions map
	schema3 := &base.Schema{Type: []string{"string"}}
	schema3.Extensions = nil
	assert.Equal(t, "", getFakeDataExtension(schema3))
}
