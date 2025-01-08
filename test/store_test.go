package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/plugin"
	"github.com/stretchr/testify/require"
)

func TestSystemStore(t *testing.T) {
	// Initialise store provider
	store.InitStoreProvider()

	// Start test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(w, r, "", []plugin.Plugin{}, &config.ImposterConfig{})
	}))
	defer server.Close()

	// Test cases for different store operations
	t.Run("Store Operations", func(t *testing.T) {
		storeName := "teststore"
		baseURL := fmt.Sprintf("%s/system/store/%s", server.URL, storeName)

		// Clean up store before each test
		req, err := http.NewRequest(http.MethodDelete, baseURL, nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		// Test PUT - Store single value
		t.Run("PUT single value", func(t *testing.T) {
			key := "key1"
			value := "value1"
			req, err := http.NewRequest(http.MethodPut, baseURL+"/"+key, bytes.NewBufferString(value))
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		// Test GET - Retrieve single value
		t.Run("GET single value", func(t *testing.T) {
			key := "key1"
			resp, err := http.Get(baseURL + "/" + key)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, "value1", string(body))
		})

		// Test POST - Store multiple values
		t.Run("POST multiple values", func(t *testing.T) {
			values := map[string]interface{}{
				"key2": "value2",
				"key3": map[string]interface{}{
					"nested": "value3",
				},
			}
			data, err := json.Marshal(values)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewBuffer(data))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		// Test GET - List all values
		t.Run("GET all values", func(t *testing.T) {
			resp, err := http.Get(baseURL)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)

			var values map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&values)
			require.NoError(t, err)

			// Check all values are present
			require.Equal(t, "value1", values["key1"])
			require.Equal(t, "value2", values["key2"])
			require.Equal(t, "value3", values["key3"].(map[string]interface{})["nested"])
		})

		// Test GET - List values with prefix
		t.Run("GET values with prefix", func(t *testing.T) {
			// First ensure key1 exists
			key := "key1"
			value := "value1"
			req, err := http.NewRequest(http.MethodPut, baseURL+"/"+key, bytes.NewBufferString(value))
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusNoContent, resp.StatusCode)

			// Add another key with different prefix
			key = "other1"
			value = "othervalue"
			req, err = http.NewRequest(http.MethodPut, baseURL+"/"+key, bytes.NewBufferString(value))
			require.NoError(t, err)

			resp, err = http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusNoContent, resp.StatusCode)

			// Now get values with prefix
			resp, err = http.Get(baseURL + "?keyPrefix=key")
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode)

			var values map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&values)
			require.NoError(t, err)

			// Check only key1 is present (without the prefix)
			value1, ok := values["1"] // key1 -> 1 after prefix removal
			require.True(t, ok, "key1 value should be present in the response")
			require.Equal(t, "value1", value1)
			require.NotContains(t, values, "other1")
		})

		// Test DELETE - Single value
		t.Run("DELETE single value", func(t *testing.T) {
			key := "key1"
			req, err := http.NewRequest(http.MethodDelete, baseURL+"/"+key, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusNoContent, resp.StatusCode)

			// Verify value is deleted
			resp, err = http.Get(baseURL + "/" + key)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		// Test DELETE - Entire store
		t.Run("DELETE entire store", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodDelete, baseURL, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, http.StatusNoContent, resp.StatusCode)

			// Verify store is empty
			resp, err = http.Get(baseURL)
			require.NoError(t, err)
			defer resp.Body.Close()

			var values map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&values)
			require.NoError(t, err)
			require.Empty(t, values)
		})

		// Test error cases
		t.Run("Error cases", func(t *testing.T) {
			// PUT without key
			t.Run("PUT without key", func(t *testing.T) {
				req, err := http.NewRequest(http.MethodPut, baseURL, bytes.NewBufferString("value"))
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})

			// POST with invalid JSON
			t.Run("POST invalid JSON", func(t *testing.T) {
				req, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewBufferString("invalid json"))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/json")

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})

			// Invalid method
			t.Run("Invalid method", func(t *testing.T) {
				req, err := http.NewRequest(http.MethodPatch, baseURL, nil)
				require.NoError(t, err)

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
			})
		})
	})
}
