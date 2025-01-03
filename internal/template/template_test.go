package template

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestProcessTemplate(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		setupRequest   func() *http.Request
		imposterConfig *config.ImposterConfig
		requestStore   store.Store
		want           string
	}{
		{
			name:     "request method",
			template: "Method: ${context.request.method}",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("POST", "/", nil)
				req.Body = io.NopCloser(strings.NewReader(""))
				return req
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.Store{},
			want:           "Method: POST",
		},
		{
			name:     "query parameters",
			template: "Hello ${context.request.queryParams.name}!",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/?name=World", nil)
				req.Body = io.NopCloser(strings.NewReader(""))
				return req
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.Store{},
			want:           "Hello World!",
		},
		{
			name:     "request headers",
			template: "User-Agent: ${context.request.headers.User-Agent}",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Set("User-Agent", "test-agent")
				req.Body = io.NopCloser(strings.NewReader(""))
				return req
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.Store{},
			want:           "User-Agent: test-agent",
		},
		{
			name:     "form parameters",
			template: "Form value: ${context.request.formParams.key}",
			setupRequest: func() *http.Request {
				body := strings.NewReader("key=value")
				req, _ := http.NewRequest("POST", "/", body)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.Store{},
			want:           "Form value: value",
		},
		{
			name:     "request body",
			template: "Body: ${context.request.body}",
			setupRequest: func() *http.Request {
				body := strings.NewReader("test body")
				req, _ := http.NewRequest("POST", "/", body)
				return req
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.Store{},
			want:           "Body: test body",
		},
		{
			name:     "request path and uri",
			template: "Path: ${context.request.path}, URI: ${context.request.uri}",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/test?param=value", nil)
				req.Body = io.NopCloser(strings.NewReader(""))
				return req
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.Store{},
			want:           "Path: /test, URI: /test?param=value",
		},
		{
			name:     "system server placeholders",
			template: "Port: ${system.server.port}",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Host = "localhost:8080"
				req.Body = io.NopCloser(strings.NewReader(""))
				return req
			},
			imposterConfig: &config.ImposterConfig{ServerPort: "8080"},
			requestStore:   store.Store{},
			want:           "Port: 8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProcessTemplate(tt.template, tt.setupRequest(), tt.imposterConfig, tt.requestStore)
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
			req.Body = io.NopCloser(strings.NewReader(""))
			imposterConfig := &config.ImposterConfig{ServerPort: "8080"}
			result := ProcessTemplate(tt.template, req, imposterConfig, store.Store{})
			tt.validate(t, result)
		})
	}
}

func TestStoreValueReplacement(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		requestStore store.Store
		want         string
	}{
		{
			name:         "request store value",
			template:     "${stores.request.key}",
			requestStore: store.Store{"key": "value"},
			want:         "value",
		},
		{
			name:         "missing store value",
			template:     "${stores.request.missing}",
			requestStore: store.Store{},
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.Body = io.NopCloser(strings.NewReader(""))
			imposterConfig := &config.ImposterConfig{ServerPort: "8080"}
			got := ProcessTemplate(tt.template, req, imposterConfig, tt.requestStore)
			assert.Equal(t, tt.want, got)
		})
	}
}
