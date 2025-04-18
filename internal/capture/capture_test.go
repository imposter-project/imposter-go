package capture

import (
	"bytes"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"net/http"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestCaptureRequestData(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }
	tests := []struct {
		name           string
		resource       config.Resource
		setupRequest   func() (*http.Request, *config.RequestMatcher, []byte)
		imposterConfig *config.ImposterConfig
		validate       func(t *testing.T, requestStore *store.Store)
	}{
		{
			name: "capture query parameter",
			resource: config.Resource{
				BaseResource: config.BaseResource{
					Capture: map[string]config.Capture{
						"test": {
							Enabled: boolPtr(true),
							Key: config.CaptureConfig{
								Const: "myKey",
							},
							CaptureConfig: config.CaptureConfig{
								QueryParam: "param",
							},
							Store: "request",
						},
					},
				},
			},
			setupRequest: func() (*http.Request, *config.RequestMatcher, []byte) {
				req, _ := http.NewRequest("GET", "/?param=value", nil)
				return req, &config.RequestMatcher{}, nil
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore *store.Store) {
				val, _ := requestStore.GetValue("myKey")
				assert.Equal(t, "value", val)
			},
		},
		{
			name: "capture path parameter",
			resource: config.Resource{
				BaseResource: config.BaseResource{
					Capture: map[string]config.Capture{
						"test": {
							Enabled: boolPtr(true),
							Key: config.CaptureConfig{
								Const: "myKey",
							},
							CaptureConfig: config.CaptureConfig{
								PathParam: "param",
							},
							Store: "request",
						},
					},
				},
			},
			setupRequest: func() (*http.Request, *config.RequestMatcher, []byte) {
				req, _ := http.NewRequest("GET", "/value", nil)
				return req, &config.RequestMatcher{Path: "/{param}"}, nil
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore *store.Store) {
				val, _ := requestStore.GetValue("myKey")
				assert.Equal(t, "value", val)
			},
		},
		{
			name: "capture request header",
			resource: config.Resource{
				BaseResource: config.BaseResource{
					Capture: map[string]config.Capture{
						"test": {
							Enabled: boolPtr(true),
							Key: config.CaptureConfig{
								Const: "headerValue",
							},
							CaptureConfig: config.CaptureConfig{
								RequestHeader: "X-Test-Header",
							},
							Store: "request",
						},
					},
				},
			},
			setupRequest: func() (*http.Request, *config.RequestMatcher, []byte) {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Set("X-Test-Header", "test-value")
				return req, &config.RequestMatcher{}, nil
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore *store.Store) {
				val, _ := requestStore.GetValue("headerValue")
				assert.Equal(t, "test-value", val)
			},
		},
		{
			name: "capture form parameter",
			resource: config.Resource{
				BaseResource: config.BaseResource{
					Capture: map[string]config.Capture{
						"test": {
							Enabled: boolPtr(true),
							Key: config.CaptureConfig{
								Const: "formValue",
							},
							CaptureConfig: config.CaptureConfig{
								FormParam: "field",
							},
							Store: "request",
						},
					},
				},
			},
			setupRequest: func() (*http.Request, *config.RequestMatcher, []byte) {
				body := strings.NewReader("field=form-data")
				req, _ := http.NewRequest("POST", "/", body)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req, &config.RequestMatcher{}, []byte("field=form-data")
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore *store.Store) {
				val, _ := requestStore.GetValue("formValue")
				assert.Equal(t, "form-data", val)
			},
		},
		{
			name: "capture JSON path",
			resource: config.Resource{
				BaseResource: config.BaseResource{
					Capture: map[string]config.Capture{
						"test": {
							Enabled: boolPtr(true),
							Key: config.CaptureConfig{
								Const: "jsonValue",
							},
							CaptureConfig: config.CaptureConfig{
								RequestBody: struct {
									JSONPath      string            `yaml:"jsonPath,omitempty"`
									XPath         string            `yaml:"xPath,omitempty"`
									XMLNamespaces map[string]string `yaml:"xmlNamespaces,omitempty"`
								}{
									JSONPath: "$.name",
								},
							},
							Store: "request",
						},
					},
				},
			},
			setupRequest: func() (*http.Request, *config.RequestMatcher, []byte) {
				jsonBody := []byte(`{"name": "test-name"}`)
				req, _ := http.NewRequest("POST", "/", bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				return req, &config.RequestMatcher{}, jsonBody
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore *store.Store) {
				val, _ := requestStore.GetValue("jsonValue")
				assert.Equal(t, "test-name", val)
			},
		},
		{
			name: "capture XML path",
			resource: config.Resource{
				BaseResource: config.BaseResource{
					Capture: map[string]config.Capture{
						"test": {
							Enabled: boolPtr(true),
							Key: config.CaptureConfig{
								Const: "xmlValue",
							},
							CaptureConfig: config.CaptureConfig{
								RequestBody: struct {
									JSONPath      string            `yaml:"jsonPath,omitempty"`
									XPath         string            `yaml:"xPath,omitempty"`
									XMLNamespaces map[string]string `yaml:"xmlNamespaces,omitempty"`
								}{
									XPath: "//name",
								},
							},
							Store: "request",
						},
					},
				},
			},
			setupRequest: func() (*http.Request, *config.RequestMatcher, []byte) {
				xmlBody := []byte(`<?xml version="1.0" encoding="UTF-8"?><root><name>test-name</name></root>`)
				req, _ := http.NewRequest("POST", "/", bytes.NewReader(xmlBody))
				req.Header.Set("Content-Type", "application/xml")
				return req, &config.RequestMatcher{}, xmlBody
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore *store.Store) {
				val, _ := requestStore.GetValue("xmlValue")
				assert.Equal(t, "test-name", val)
			},
		},
		{
			name: "capture disabled",
			resource: config.Resource{
				BaseResource: config.BaseResource{
					Capture: map[string]config.Capture{
						"test": {
							Enabled: boolPtr(false),
							Key: config.CaptureConfig{
								Const: "disabled",
							},
							CaptureConfig: config.CaptureConfig{
								QueryParam: "param",
							},
							Store: "request",
						},
					},
				},
			},
			setupRequest: func() (*http.Request, *config.RequestMatcher, []byte) {
				req, _ := http.NewRequest("GET", "/?param=value", nil)
				return req, &config.RequestMatcher{}, nil
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore *store.Store) {
				_, exists := requestStore.GetValue("disabled")
				assert.False(t, exists)
			},
		},
		{
			name: "capture enabled not set",
			resource: config.Resource{
				BaseResource: config.BaseResource{
					Capture: map[string]config.Capture{
						"test": {
							Key: config.CaptureConfig{
								Const: "enabled_not_set",
							},
							CaptureConfig: config.CaptureConfig{
								QueryParam: "param",
							},
							Store: "request",
						},
					},
				},
			},
			setupRequest: func() (*http.Request, *config.RequestMatcher, []byte) {
				req, _ := http.NewRequest("GET", "/?param=value", nil)
				return req, &config.RequestMatcher{}, nil
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore *store.Store) {
				val, _ := requestStore.GetValue("enabled_not_set")
				assert.Equal(t, "value", val)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestStore := store.NewRequestStore()
			req, reqMatcher, body := tt.setupRequest()
			exch := exchange.NewExchangeFromRequest(req, body, requestStore)
			CaptureRequestData(tt.imposterConfig, reqMatcher, tt.resource.Capture, exch)
			tt.validate(t, requestStore)
		})
	}
}
