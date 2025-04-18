package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
	"github.com/imposter-project/imposter-go/plugin"
	"github.com/stretchr/testify/require"
)

func TestSystemStatus(t *testing.T) {
	// Create test configuration
	configs := []config.Config{
		{
			Plugin: "rest",
			Resources: []config.Resource{
				{
					BaseResource: config.BaseResource{
						RequestMatcher: config.RequestMatcher{
							Path: "/test",
						},
						Response: &config.Response{
							Content:    "test response",
							StatusCode: 200,
						},
					},
				},
			},
		},
	}
	imposterConfig := &config.ImposterConfig{
		ServerPort: "8080",
	}
	plugins := plugin.LoadPlugins(configs, "", imposterConfig)

	// Start test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(nil, w, r, plugins)
	}))
	defer server.Close()

	// Test cases
	tests := []struct {
		name           string
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:           "Get system status",
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status":  "ok",
				"version": "dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make request to /system/status
			resp, err := http.Get(server.URL + "/system/status")
			require.NoError(t, err)
			defer resp.Body.Close()

			// Check status code
			require.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err)

			// Check response body
			require.Equal(t, tt.expectedBody, body)
		})
	}
}
