package steps

import (
	"net/http"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestExecuteScriptStep(t *testing.T) {
	tests := []struct {
		name        string
		step        config.Step
		setupExch   func() *exchange.Exchange
		setupStore  func()
		validate    func(t *testing.T)
		expectError bool
	}{
		{
			name: "inline script accessing request context",
			step: config.Step{
				Type: config.ScriptStepType,
				Lang: "javascript",
				Code: `
					if (context.request.method !== "POST") {
						throw new Error("Expected POST method");
					}
					if (context.request.path !== "/test") {
						throw new Error("Expected /test path");
					}
					if (context.request.body !== "test body") {
						throw new Error("Expected test body");
					}
					if (context.request.queryParams.foo !== "bar") {
						throw new Error("Expected foo=bar query param");
					}
					if (context.request.headers["X-Test"] !== "test-value") {
						throw new Error("Expected X-Test header");
					}
				`,
			},
			setupExch: func() *exchange.Exchange {
				req, _ := http.NewRequest("POST", "/test?foo=bar", strings.NewReader("test body"))
				req.Header.Set("X-Test", "test-value")
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte("test body"),
					},
				}
			},
		},
		{
			name: "store operations",
			step: config.Step{
				Type: config.ScriptStepType,
				Lang: "javascript",
				Code: `
					var testStore = stores.open("test");
					
					// Test save and load
					testStore.save("key1", "value1");
					if (testStore.load("key1") !== "value1") {
						throw new Error("Failed to load saved value");
					}
					
					// Test JSON operations
					testStore.save("key2", { foo: "bar" });
					var json = testStore.loadAsJson("key2");
					if (json.foo !== "bar") {
						throw new Error("Failed to load JSON value");
					}
					
					// Test hasItemWithKey
					if (!testStore.hasItemWithKey("key1")) {
						throw new Error("hasItemWithKey failed for existing key");
					}
					if (testStore.hasItemWithKey("nonexistent")) {
						throw new Error("hasItemWithKey failed for non-existent key");
					}
					
					// Test loadAll
					var all = testStore.loadAll();
					if (!all.key1 || !all.key2) {
						throw new Error("loadAll failed to return all keys");
					}
					
					// Test delete
					testStore.delete("key1");
					if (testStore.hasItemWithKey("key1")) {
						throw new Error("delete failed");
					}
				`,
			},
			setupExch: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/test", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte{},
					},
				}
			},
			setupStore: func() {
				store.InitStoreProvider()
			},
		},
		{
			name: "invalid language",
			step: config.Step{
				Type: config.ScriptStepType,
				Lang: "groovy",
				Code: "println 'hello'",
			},
			setupExch: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/test", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte{},
					},
				}
			},
			expectError: true,
		},
		{
			name: "missing code and file",
			step: config.Step{
				Type: config.ScriptStepType,
				Lang: "javascript",
			},
			setupExch: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/test", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte{},
					},
				}
			},
			expectError: true,
		},
		{
			name: "script syntax error",
			step: config.Step{
				Type: config.ScriptStepType,
				Lang: "javascript",
				Code: "this is not valid javascript;",
			},
			setupExch: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/test", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte{},
					},
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupStore != nil {
				tt.setupStore()
			}

			exch := tt.setupExch()
			err := executeScriptStep(&tt.step, exch)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t)
			}
		})
	}
}
