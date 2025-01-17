package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIHandlerEndToEnd(t *testing.T) {
	tests := []struct {
		name          string
		configDir     string
		request       *http.Request
		wantStatus    int
		wantBodyJson  bool
		wantBodyMatch string
		wantHeaders   map[string]string
	}{
		{
			name:      "OpenAPI 3.0 - Get Pet by ID",
			configDir: "testdata/v30",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/pet/123", nil)
				req.Header.Set("Accept", "application/json")
				return req
			}(),
			wantStatus:   http.StatusOK,
			wantBodyJson: true,
			wantBodyMatch: `{
  "category" : {
    "id" : "1",
    "name" : "Dogs"
  },
  "id" : "10",
  "name" : "doggie",
  "photoUrls" : [ "example" ],
  "status" : "available",
  "tags" : [ {
    "id" : 42,
    "name" : "example"
  } ]
}`,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:      "OpenAPI 3.0 - Add New Pet",
			configDir: "testdata/v30",
			request: func() *http.Request {
				body := strings.NewReader(`{"id": 999, "name": "TestPet", "status": "available"}`)
				req := httptest.NewRequest(http.MethodPost, "/pet", body)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json")
				return req
			}(),
			wantStatus:   http.StatusOK,
			wantBodyJson: true,
			wantBodyMatch: `{
  "category" : {
    "id" : "1",
    "name" : "Dogs"
  },
  "id" : "10",
  "name" : "doggie",
  "photoUrls" : [ "example" ],
  "status" : "available",
  "tags" : [ {
    "id" : 42,
    "name" : "example"
  } ]
}`,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:      "OpenAPI 3.0 - Invalid Pet ID",
			configDir: "testdata/v30",
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/pet/invalid", nil)
			}(),
			wantStatus:    http.StatusBadRequest,
			wantBodyJson:  false,
			wantBodyMatch: ``,
		},
		{
			name:      "OpenAPI 3.0 - Pet Not Found",
			configDir: "testdata/v30",
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/pet/99999", nil)
			}(),
			wantStatus:    http.StatusNotFound,
			wantBodyJson:  false,
			wantBodyMatch: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize store
			store.InitStoreProvider()

			// Load config
			configs := config.LoadConfig(tt.configDir)
			require.Len(t, configs, 1, "Expected one config")
			cfg := &configs[0]

			// Create handler
			handler, err := NewPluginHandler(cfg, tt.configDir, &config.ImposterConfig{})
			require.NoError(t, err, "Failed to create handler")

			// Create response recorder
			responseState := response.NewResponseState()
			requestStore := make(store.Store)

			// Handle request
			handler.HandleRequest(tt.request, requestStore, responseState)

			// Assert status code
			assert.Equal(t, tt.wantStatus, responseState.StatusCode, "Unexpected status code")

			// Assert response body
			if tt.wantBodyMatch != "" {
				if tt.wantBodyJson {
					assert.JSONEq(t, string(responseState.Body), tt.wantBodyMatch, "JSON response body mismatch")
				} else {
					assert.Contains(t, string(responseState.Body), tt.wantBodyMatch, "Response body mismatch")
				}
			}

			// Assert headers
			for k, v := range tt.wantHeaders {
				assert.Equal(t, v, responseState.Headers[k], "Header mismatch for %s", k)
			}

			// Validate response against OpenAPI schema if it's a successful response
			if tt.wantStatus == http.StatusOK {
				var respBody map[string]interface{}
				err = json.Unmarshal(responseState.Body, &respBody)
				require.NoError(t, err, "Failed to parse response body")

				// TODO: Add schema validation once implemented
				// err = handler.openApiParser.ValidateResponse(tt.request.URL.Path, tt.request.Method, tt.wantStatus, respBody)
				// assert.NoError(t, err, "Response schema validation failed")
			}
		})
	}
}

func TestOpenAPIHandlerValidation(t *testing.T) {
	tests := []struct {
		name          string
		configDir     string
		request       *http.Request
		wantStatus    int
		wantBodyMatch string
	}{
		{
			name:      "Invalid Request Body Schema",
			configDir: "testdata/v30",
			request: func() *http.Request {
				body := strings.NewReader(`{"id": "invalid", "status": 123}`) // id should be number, status should be string
				req := httptest.NewRequest(http.MethodPost, "/pet", body)
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
			wantStatus:    http.StatusBadRequest,
			wantBodyMatch: `"message":"Invalid request body"`,
		},
		{
			name:      "Missing Required Field",
			configDir: "testdata/v30",
			request: func() *http.Request {
				body := strings.NewReader(`{"status": "available"}`) // missing required 'id' field
				req := httptest.NewRequest(http.MethodPost, "/pet", body)
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
			wantStatus:    http.StatusBadRequest,
			wantBodyMatch: `"message":"Missing required field: id"`,
		},
		{
			name:      "Invalid Content Type",
			configDir: "testdata/v30",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/pet", strings.NewReader(`{}`))
				req.Header.Set("Content-Type", "text/plain")
				return req
			}(),
			wantStatus:    http.StatusUnsupportedMediaType,
			wantBodyMatch: `"message":"Unsupported Media Type"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize store
			store.InitStoreProvider()

			// Load config
			configs := config.LoadConfig(tt.configDir)
			require.Len(t, configs, 1, "Expected one config")
			cfg := &configs[0]

			// Create handler
			handler, err := NewPluginHandler(cfg, tt.configDir, nil)
			require.NoError(t, err, "Failed to create handler")

			// Create response recorder
			responseState := &response.ResponseState{}
			requestStore := store.Store{}

			// Handle request
			handler.HandleRequest(tt.request, requestStore, responseState)

			// Assert status code
			assert.Equal(t, tt.wantStatus, responseState.StatusCode, "Unexpected status code")

			// Assert response body
			if tt.wantBodyMatch != "" {
				assert.Contains(t, responseState.Body, tt.wantBodyMatch, "Response body mismatch")
			}
		})
	}
}
