package template

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestProcessTemplate(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		setupRequest   func() (*http.Request, string, *config.RequestMatcher)
		imposterConfig *config.ImposterConfig
		requestStore   *store.Store
		want           string
	}{
		{
			name:     "request method",
			template: "Method: ${context.request.method}",
			setupRequest: func() (*http.Request, string, *config.RequestMatcher) {
				req, _ := http.NewRequest("POST", "/", nil)
				return req, "", &config.RequestMatcher{}
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.NewRequestStore(),
			want:           "Method: POST",
		},
		{
			name:     "query parameters",
			template: "Hello ${context.request.queryParams.name}!",
			setupRequest: func() (*http.Request, string, *config.RequestMatcher) {
				req, _ := http.NewRequest("GET", "/?name=World", nil)
				return req, "", &config.RequestMatcher{}
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.NewRequestStore(),
			want:           "Hello World!",
		},
		{
			name:     "path parameters",
			template: "Hello ${context.request.pathParams.name}!",
			setupRequest: func() (*http.Request, string, *config.RequestMatcher) {
				req, _ := http.NewRequest("GET", "/param", nil)
				return req, "", &config.RequestMatcher{Path: "/{name}"}
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.NewRequestStore(),
			want:           "Hello param!",
		},
		{
			name:     "request headers",
			template: "User-Agent: ${context.request.headers.User-Agent}",
			setupRequest: func() (*http.Request, string, *config.RequestMatcher) {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Set("User-Agent", "test-agent")
				return req, "", &config.RequestMatcher{}
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.NewRequestStore(),
			want:           "User-Agent: test-agent",
		},
		{
			name:     "form parameters",
			template: "Form value: ${context.request.formParams.key}",
			setupRequest: func() (*http.Request, string, *config.RequestMatcher) {
				body := "key=value"
				req, _ := http.NewRequest("POST", "/", nil)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req, body, &config.RequestMatcher{}
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.NewRequestStore(),
			want:           "Form value: value",
		},
		{
			name:     "request body",
			template: "Body: ${context.request.body}",
			setupRequest: func() (*http.Request, string, *config.RequestMatcher) {
				body := "test body"
				req, _ := http.NewRequest("POST", "/", nil)
				return req, body, &config.RequestMatcher{}
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.NewRequestStore(),
			want:           "Body: test body",
		},
		{
			name:     "request path and uri",
			template: "Path: ${context.request.path}, URI: ${context.request.uri}",
			setupRequest: func() (*http.Request, string, *config.RequestMatcher) {
				req, _ := http.NewRequest("GET", "/test?param=value", nil)
				return req, "", &config.RequestMatcher{}
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.NewRequestStore(),
			want:           "Path: /test, URI: /test?param=value",
		},
		{
			name:     "system server port placeholder",
			template: "Port: ${system.server.port}",
			setupRequest: func() (*http.Request, string, *config.RequestMatcher) {
				req, _ := http.NewRequest("GET", "/", nil)
				return req, "", &config.RequestMatcher{}
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.NewRequestStore(),
			want:           "Port: 8080",
		},
		{
			name:     "system server url placeholder",
			template: "Server URL: ${system.server.url}",
			setupRequest: func() (*http.Request, string, *config.RequestMatcher) {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Body = io.NopCloser(strings.NewReader(""))
				return req, "", &config.RequestMatcher{}
			},
			imposterConfig: &config.ImposterConfig{ServerUrl: "http://localhost:8080"},
			requestStore:   store.NewRequestStore(),
			want:           "Server URL: http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, body, reqMatcher := tt.setupRequest()
			req.Body = io.NopCloser(strings.NewReader(body))
			exch := exchange.NewExchangeFromRequest(req, []byte(body), tt.requestStore)
			got := ProcessTemplate(tt.template, exch, tt.imposterConfig, reqMatcher)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRandomPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		template string
		validate func(t *testing.T, result string)
	}{
		{
			name:     "alphabetic",
			template: "${random.alphabetic(length=5,uppercase=true)}",
			validate: func(t *testing.T, result string) {
				assert.Len(t, result, 5)
				assert.Regexp(t, "^[A-Z]{5}$", result)
			},
		},
		{
			name:     "alphanumeric",
			template: "${random.alphanumeric(length=8)}",
			validate: func(t *testing.T, result string) {
				assert.Len(t, result, 8)
				assert.Regexp(t, "^[a-zA-Z0-9]{8}$", result)
			},
		},
		{
			name:     "numeric",
			template: "${random.numeric(length=3)}",
			validate: func(t *testing.T, result string) {
				assert.Len(t, result, 3)
				assert.Regexp(t, "^[0-9]{3}$", result)
			},
		},
		{
			name:     "uuid",
			template: "${random.uuid()}",
			validate: func(t *testing.T, result string) {
				assert.Regexp(t, "^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$", result)
			},
		},
		{
			name:     "any with custom chars",
			template: "${random.any(chars=abc123,length=4)}",
			validate: func(t *testing.T, result string) {
				assert.Len(t, result, 4)
				assert.Regexp(t, "^[abc123]{4}$", result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			body := []byte("")
			req.Body = io.NopCloser(bytes.NewReader(body))
			imposterConfig := &config.ImposterConfig{ServerPort: "8080"}
			exch := exchange.NewExchangeFromRequest(req, body, store.NewRequestStore())
			result := ProcessTemplate(tt.template, exch, imposterConfig, &config.RequestMatcher{})
			tt.validate(t, result)
		})
	}
}

func TestStoreValueReplacement(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		requestStore func() *store.Store
		want         string
	}{
		{
			name:     "request store value",
			template: "${stores.request.key}",
			requestStore: func() *store.Store {
				s := store.NewRequestStore()
				s.StoreValue("key", "value")
				return s
			},
			want: "value",
		},
		{
			name:     "missing store value",
			template: "${stores.request.missing}",
			requestStore: func() *store.Store {
				s := store.NewRequestStore()
				return s
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			body := []byte("")
			req.Body = io.NopCloser(bytes.NewReader(body))
			imposterConfig := &config.ImposterConfig{ServerPort: "8080"}
			exch := exchange.NewExchangeFromRequest(req, body, tt.requestStore())
			got := ProcessTemplate(tt.template, exch, imposterConfig, &config.RequestMatcher{})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResponseTemplates(t *testing.T) {
	tests := []struct {
		name     string
		template string
		setup    func() *exchange.Exchange
		want     string
	}{
		{
			name:     "response body",
			template: "Response: ${context.response.body}",
			setup: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte(""),
					},
					Response: &exchange.ResponseContext{
						Response: &http.Response{StatusCode: 200},
						Body:     []byte(`{"message":"success"}`),
					},
				}
			},
			want: `Response: {"message":"success"}`,
		},
		{
			name:     "response headers",
			template: "Content-Type: ${context.response.headers.Content-Type}",
			setup: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/", nil)
				resp := &http.Response{StatusCode: 200, Header: http.Header{}}
				resp.Header.Set("Content-Type", "application/json")
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte(""),
					},
					Response: &exchange.ResponseContext{
						Response: resp,
						Body:     []byte{},
					},
				}
			},
			want: "Content-Type: application/json",
		},
		{
			name:     "response status code",
			template: "Status: ${context.response.statusCode}",
			setup: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte(""),
					},
					Response: &exchange.ResponseContext{
						Response: &http.Response{StatusCode: 201},
						Body:     []byte{},
					},
				}
			},
			want: "Status: 201",
		},
		{
			name:     "response body with JSONPath",
			template: "Message: ${context.response.body:$.message}",
			setup: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte(""),
					},
					Response: &exchange.ResponseContext{
						Response: &http.Response{StatusCode: 200},
						Body:     []byte(`{"message":"success"}`),
					},
				}
			},
			want: "Message: success",
		},
		{
			name:     "response body with XPath",
			template: "Name: ${context.response.body:/root/name}",
			setup: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte(""),
					},
					Response: &exchange.ResponseContext{
						Response: &http.Response{StatusCode: 200},
						Body:     []byte(`<root><name>test</name></root>`),
					},
				}
			},
			want: "Name: test",
		},
		{
			name:     "response header with default value",
			template: "${context.response.headers.X-Missing:-default}",
			setup: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte(""),
					},
					Response: &exchange.ResponseContext{
						Response: &http.Response{StatusCode: 200, Header: http.Header{}},
						Body:     []byte{},
					},
				}
			},
			want: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exch := tt.setup()
			got := ProcessTemplate(tt.template, exch, nil, &config.RequestMatcher{})
			assert.Equal(t, tt.want, got)
		})
	}
}
