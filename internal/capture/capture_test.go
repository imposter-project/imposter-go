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
		setupRequest   func() (*http.Request, []byte)
		imposterConfig *config.ImposterConfig
		validate       func(t *testing.T, requestStore store.Store)
	}{
		{
			name: "capture query parameter",
			resource: config.Resource{
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
			setupRequest: func() (*http.Request, []byte) {
				req, _ := http.NewRequest("GET", "/?param=value", nil)
				return req, nil
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore store.Store) {
				assert.Equal(t, "value", requestStore["myKey"])
			},
		},
		{
			name: "capture request header",
			resource: config.Resource{
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
			setupRequest: func() (*http.Request, []byte) {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Set("X-Test-Header", "test-value")
				return req, nil
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore store.Store) {
				assert.Equal(t, "test-value", requestStore["headerValue"])
			},
		},
		{
			name: "capture form parameter",
			resource: config.Resource{
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
			setupRequest: func() (*http.Request, []byte) {
				body := strings.NewReader("field=form-data")
				req, _ := http.NewRequest("POST", "/", body)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req, []byte("field=form-data")
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore store.Store) {
				assert.Equal(t, "form-data", requestStore["formValue"])
			},
		},
		{
			name: "capture JSON path",
			resource: config.Resource{
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
			setupRequest: func() (*http.Request, []byte) {
				jsonBody := []byte(`{"name": "test-name"}`)
				req, _ := http.NewRequest("POST", "/", bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				return req, jsonBody
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore store.Store) {
				assert.Equal(t, "test-name", requestStore["jsonValue"])
			},
		},
		{
			name: "capture XML path",
			resource: config.Resource{
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
			setupRequest: func() (*http.Request, []byte) {
				xmlBody := []byte(`<?xml version="1.0" encoding="UTF-8"?><root><name>test-name</name></root>`)
				req, _ := http.NewRequest("POST", "/", bytes.NewReader(xmlBody))
				req.Header.Set("Content-Type", "application/xml")
				return req, xmlBody
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore store.Store) {
				assert.Equal(t, "test-name", requestStore["xmlValue"])
			},
		},
		{
			name: "capture disabled",
			resource: config.Resource{
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
			setupRequest: func() (*http.Request, []byte) {
				req, _ := http.NewRequest("GET", "/?param=value", nil)
				return req, nil
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore store.Store) {
				_, exists := requestStore["disabled"]
				assert.False(t, exists)
			},
		},
		{
			name: "capture enabled not set",
			resource: config.Resource{
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
			setupRequest: func() (*http.Request, []byte) {
				req, _ := http.NewRequest("GET", "/?param=value", nil)
				return req, nil
			},
			imposterConfig: &config.ImposterConfig{},
			validate: func(t *testing.T, requestStore store.Store) {
				assert.Equal(t, "value", requestStore["enabled_not_set"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestStore := store.Store{}
			req, body := tt.setupRequest()
			exch := exchange.NewExchangeFromRequest(req, body, &requestStore)
			CaptureRequestData(tt.imposterConfig, tt.resource.Capture, exch)
			tt.validate(t, requestStore)
		})
	}
}
