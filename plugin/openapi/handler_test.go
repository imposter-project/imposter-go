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
		// OpenAPI 3.0 Tests
		{
			name:      "OpenAPI 3.0 - Get Pet by ID - default example",
			configDir: "testdata/v30",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/v3/pet/123", nil)
				req.Header.Set("Accept", "application/json")
				return req
			}(),
			wantStatus:   http.StatusOK,
			wantBodyJson: true,
			wantBodyMatch: `{
  "category" : {
    "id" : 1,
    "name" : "Cats"
  },
  "id" : 200,
  "name" : "fluffy",
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
			name:      "OpenAPI 3.0 - Add New Pet - generate from schema",
			configDir: "testdata/v30",
			request: func() *http.Request {
				body := strings.NewReader(`{"id": 999, "name": "TestPet", "status": "available"}`)
				req := httptest.NewRequest(http.MethodPost, "/v3/pet", body)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json")
				return req
			}(),
			wantStatus:   http.StatusOK,
			wantBodyJson: true,
			wantBodyMatch: `{
  "category" : {
    "id" : 1,
    "name" : "Dogs"
  },
  "id" : 10,
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
				return httptest.NewRequest(http.MethodGet, "/v3/pet/invalid", nil)
			}(),
			wantStatus:    http.StatusBadRequest,
			wantBodyJson:  false,
			wantBodyMatch: ``,
		},
		{
			name:      "OpenAPI 3.0 - Pet Not Found",
			configDir: "testdata/v30",
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v3/pet/99999", nil)
			}(),
			wantStatus:    http.StatusNotFound,
			wantBodyJson:  false,
			wantBodyMatch: ``,
		},
		{
			name:      "OpenAPI 3.0 - Named example",
			configDir: "testdata/v30",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/v3/pet/100", nil)
				req.Header.Set("Accept", "application/json")
				return req
			}(),
			wantStatus:   http.StatusOK,
			wantBodyJson: true,
			wantBodyMatch: `{
  "category" : {
    "id" : 2,
    "name" : "Dogs"
  },
  "id" : 100,
  "name" : "woof",
  "photoUrls" : [ "example" ],
  "status" : "available",
  "tags" : [ {
    "id" : 43,
    "name" : "example"
  } ]
}`,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		// OpenAPI 2.0 Tests
		{
			name:      "OpenAPI 2.0 - Get Pet by ID",
			configDir: "testdata/v20",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/v2/pet/123", nil)
				req.Header.Set("Accept", "application/json")
				return req
			}(),
			wantStatus:   http.StatusOK,
			wantBodyJson: true,
			wantBodyMatch: `{
  "id": 42,
  "category": {
    "id": 42,
    "name": "example"
  },
  "name": "doggie",
  "photoUrls": [
    "example"
  ],
  "tags": [
    {
      "id": 42,
      "name": "example"
    }
  ],
  "status": "available"
}`,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:      "OpenAPI 2.0 - Add New Pet",
			configDir: "testdata/v20",
			request: func() *http.Request {
				body := strings.NewReader(`{
					"id": 999,
					"name": "TestPet",
					"status": "available",
					"photoUrls": ["http://example.com/photo.jpg"]
				}`)
				req := httptest.NewRequest(http.MethodPost, "/v2/pet", body)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json")
				return req
			}(),
			wantStatus:    http.StatusMethodNotAllowed,
			wantBodyJson:  false,
			wantBodyMatch: ``,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:      "OpenAPI 2.0 - Invalid Pet ID",
			configDir: "testdata/v20",
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v2/pet/invalid", nil)
			}(),
			wantStatus:    http.StatusBadRequest,
			wantBodyJson:  false,
			wantBodyMatch: ``,
		},
		{
			name:      "OpenAPI 2.0 - Pet Not Found",
			configDir: "testdata/v20",
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/v2/pet/99999", nil)
			}(),
			wantStatus:    http.StatusNotFound,
			wantBodyJson:  true,
			wantBodyMatch: ``,
		},
		{
			name:      "OpenAPI 2.0 - Upload Pet Image",
			configDir: "testdata/v20",
			request: func() *http.Request {
				body := strings.NewReader(`--boundary
Content-Disposition: form-data; name="file"; filename="pet.jpg"
Content-Type: image/jpeg

<binary data here>
--boundary--`)
				req := httptest.NewRequest(http.MethodPost, "/v2/pet/123/uploadImage", body)
				req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
				req.Header.Set("Accept", "application/json")
				return req
			}(),
			wantStatus:   http.StatusOK,
			wantBodyJson: true,
			wantBodyMatch: `{
  "code": 42,
  "message": "example",
  "type": "example"
}`,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
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
			responseProc := response.NewProcessor(&config.ImposterConfig{}, tt.configDir)

			// Handle request
			handler.HandleRequest(tt.request, &requestStore, responseState, responseProc)

			// Assert status code
			assert.Equal(t, tt.wantStatus, responseState.StatusCode, "Unexpected status code")

			// Assert response body
			if tt.wantBodyMatch != "" {
				if tt.wantBodyJson {
					assert.JSONEq(t, tt.wantBodyMatch, string(responseState.Body), "JSON response body mismatch")
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
