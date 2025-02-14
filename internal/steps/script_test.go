package steps

import (
	"net/http"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestExecuteScriptStep(t *testing.T) {
	tests := []struct {
		name        string
		step        config.Step
		setupExch   func() *exchange.Exchange
		setupStore  func()
		validate    func(t *testing.T, responseState *response.ResponseState)
		expectError bool
		reqMatcher  *config.RequestMatcher
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
					if (context.request.body !== "key1=value1&key2=value2") {
						throw new Error("Expected test body");
					}
					if (context.request.queryParams.foo !== "bar") {
						throw new Error("Expected foo=bar query param");
					}
					if (context.request.pathParams['path-param'] !== "test") {
						throw new Error("Expected baz=test path param");
					}
					if (context.request.headers["X-Test"] !== "test-value") {
						throw new Error("Expected X-Test header");
					}
					if (context.request.formParams.key1 !== "value1") {
						throw new Error("Expected key1=value1 form param");
					}
					if (context.request.formParams.key2 !== "value2") {
						throw new Error("Expected key2=value2 form param");
					}
				`,
			},
			setupExch: func() *exchange.Exchange {
				body := "key1=value1&key2=value2"
				req, _ := http.NewRequest("POST", "/test?foo=bar", strings.NewReader(body))
				req.Header.Set("X-Test", "test-value")
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte(body),
					},
				}
			},
			reqMatcher: &config.RequestMatcher{
				Path: "/{path-param}",
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
			reqMatcher: &config.RequestMatcher{
				Path: "/test",
			},
		},
		{
			name: "response builder with skipDefaultBehaviour",
			step: config.Step{
				Type: config.ScriptStepType,
				Lang: "javascript",
				Code: `
					respond()
						.withStatusCode(201)
						.withContent('{"status":"created"}')
						.withHeader("X-Custom", "test")
						.skipDefaultBehaviour()
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
			validate: func(t *testing.T, rs *response.ResponseState) {
				assert.Equal(t, 201, rs.StatusCode)
				assert.Equal(t, `{"status":"created"}`, string(rs.Body))
				assert.Equal(t, "test", rs.Headers["X-Custom"])
				assert.True(t, rs.Handled, "response should be marked as handled")
			},
			reqMatcher: &config.RequestMatcher{
				Path: "/test",
			},
		},
		{
			name: "response builder with usingDefaultBehaviour",
			step: config.Step{
				Type: config.ScriptStepType,
				Lang: "javascript",
				Code: `
					respond()
						.withStatusCode(201)
						.withContent('{"status":"created"}')
						.withHeader("X-Custom", "test")
						.usingDefaultBehaviour()
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
			validate: func(t *testing.T, rs *response.ResponseState) {
				assert.Equal(t, 201, rs.StatusCode)
				assert.Equal(t, `{"status":"created"}`, string(rs.Body))
				assert.Equal(t, "test", rs.Headers["X-Custom"])
				assert.False(t, rs.Handled, "response should not be marked as handled")
			},
			reqMatcher: &config.RequestMatcher{
				Path: "/test",
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
			reqMatcher: &config.RequestMatcher{
				Path: "/test",
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
			reqMatcher: &config.RequestMatcher{
				Path: "/test",
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
			reqMatcher: &config.RequestMatcher{
				Path: "/test",
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
			responseState := response.NewResponseState()
			err := executeScriptStep(&tt.step, exch, responseState, "", tt.reqMatcher)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t, responseState)
			}
		})
	}
}
