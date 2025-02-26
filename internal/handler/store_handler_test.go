package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestHandleStoreRequest(t *testing.T) {
	// Initialize store provider
	store.InitStoreProvider()

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		headers        map[string]string
		setupStore     func()
		expectedStatus int
		expectedBody   string
		validateFunc   func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name:           "invalid path",
			method:         http.MethodGet,
			path:           "/system/store",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid store path\n",
		},
		{
			name:   "get all items - empty store",
			method: http.MethodGet,
			path:   "/system/store/test-store",
			setupStore: func() {
				store.DeleteStore("test-store")
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
				var items map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &items)
				assert.NoError(t, err)
				assert.Empty(t, items)
			},
		},
		{
			name:   "get all items - with items",
			method: http.MethodGet,
			path:   "/system/store/test-store",
			setupStore: func() {
				s := store.Open("test-store", nil)
				s.StoreValue("key1", "value1")
				s.StoreValue("key2", "value2")
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
				var items map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &items)
				assert.NoError(t, err)
				assert.Equal(t, 2, len(items))
				assert.Equal(t, "value1", items["key1"])
				assert.Equal(t, "value2", items["key2"])
			},
		},
		{
			name:   "get all items - with key prefix",
			method: http.MethodGet,
			path:   "/system/store/test-store?keyPrefix=key",
			setupStore: func() {
				s := store.Open("test-store", nil)
				s.StoreValue("key1", "value1")
				s.StoreValue("key2", "value2")
				s.StoreValue("other", "value3")
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
				var items map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &items)
				assert.NoError(t, err)
				assert.Equal(t, 2, len(items))
				assert.Equal(t, "value1", items["key1"])
				assert.Equal(t, "value2", items["key2"])
			},
		},
		{
			name:   "get all items - unsupported accept header",
			method: http.MethodGet,
			path:   "/system/store/test-store",
			headers: map[string]string{
				"Accept": "text/html",
			},
			expectedStatus: http.StatusNotAcceptable,
			expectedBody:   "This endpoint only supports application/json responses\n",
		},
		{
			name:   "get all items - accept any content type",
			method: http.MethodGet,
			path:   "/system/store/test-store",
			headers: map[string]string{
				"Accept": "*/*",
			},
			setupStore: func() {
				s := store.Open("test-store", nil)
				s.StoreValue("key1", "value1")
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
				var items map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &items)
				assert.NoError(t, err)
				assert.Equal(t, 3, len(items))
			},
		},
		{
			name:   "get specific item - string value",
			method: http.MethodGet,
			path:   "/system/store/test-store/key1",
			setupStore: func() {
				s := store.Open("test-store", nil)
				s.StoreValue("key1", "value1")
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "value1",
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, "text/plain", resp.Header().Get("Content-Type"))
			},
		},
		{
			name:   "get specific item - JSON value",
			method: http.MethodGet,
			path:   "/system/store/test-store/key-json",
			setupStore: func() {
				s := store.Open("test-store", nil)
				s.StoreValue("key-json", map[string]interface{}{"name": "test"})
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
				var data map[string]interface{}
				err := json.Unmarshal(resp.Body.Bytes(), &data)
				assert.NoError(t, err)
				assert.Equal(t, "test", data["name"])
			},
		},
		{
			name:           "get non-existent item",
			method:         http.MethodGet,
			path:           "/system/store/test-store/non-existent",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Not found\n",
		},
		{
			name:           "put without key",
			method:         http.MethodPut,
			path:           "/system/store/test-store",
			body:           "test value",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Key is required\n",
		},
		{
			name:   "put new item",
			method: http.MethodPut,
			path:   "/system/store/test-store/new-key",
			body:   "new value",
			setupStore: func() {
				store.DeleteStore("test-store")
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				s := store.Open("test-store", nil)
				value, found := s.GetValue("new-key")
				assert.True(t, found)
				assert.Equal(t, "new value", value)
			},
		},
		{
			name:   "put existing item",
			method: http.MethodPut,
			path:   "/system/store/test-store/existing-key",
			body:   "updated value",
			setupStore: func() {
				s := store.Open("test-store", nil)
				s.StoreValue("existing-key", "original value")
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				s := store.Open("test-store", nil)
				value, found := s.GetValue("existing-key")
				assert.True(t, found)
				assert.Equal(t, "updated value", value)
			},
		},
		{
			name:   "post multiple items",
			method: http.MethodPost,
			path:   "/system/store/test-store",
			body:   `{"key1":"value1","key2":"value2"}`,
			setupStore: func() {
				store.DeleteStore("test-store")
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				s := store.Open("test-store", nil)
				value1, found1 := s.GetValue("key1")
				assert.True(t, found1)
				assert.Equal(t, "value1", value1)

				value2, found2 := s.GetValue("key2")
				assert.True(t, found2)
				assert.Equal(t, "value2", value2)
			},
		},
		{
			name:           "post invalid JSON",
			method:         http.MethodPost,
			path:           "/system/store/test-store",
			body:           `{"key1":"value1"`, // Invalid JSON
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON\n",
		},
		{
			name:   "delete specific item",
			method: http.MethodDelete,
			path:   "/system/store/test-store/key-to-delete",
			setupStore: func() {
				s := store.Open("test-store", nil)
				s.StoreValue("key-to-delete", "value")
				s.StoreValue("other-key", "other-value")
			},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				s := store.Open("test-store", nil)
				_, found1 := s.GetValue("key-to-delete")
				assert.False(t, found1)

				_, found2 := s.GetValue("other-key")
				assert.True(t, found2)
			},
		},
		{
			name:   "delete entire store",
			method: http.MethodDelete,
			path:   "/system/store/test-store",
			setupStore: func() {
				s := store.Open("test-store", nil)
				s.StoreValue("key1", "value1")
				s.StoreValue("key2", "value2")
			},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, resp *httptest.ResponseRecorder) {
				s := store.Open("test-store", nil)
				items := s.GetAllValues("")
				assert.Empty(t, items)
			},
		},
		{
			name:           "method not allowed",
			method:         http.MethodPatch,
			path:           "/system/store/test-store",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup store if needed
			if tt.setupStore != nil {
				tt.setupStore()
			}

			// Create request
			req, err := http.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			assert.NoError(t, err)

			// Add headers if specified
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			HandleStoreRequest(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check body if expected
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, rr.Body.String())
			}

			// Run custom validation if provided
			if tt.validateFunc != nil {
				tt.validateFunc(t, rr)
			}
		})
	}
}

func TestHandleGetStore_ContentNegotiation(t *testing.T) {
	// Initialize store provider
	store.InitStoreProvider()

	// Setup test store with a JSON value
	s := store.Open("test-store", nil)
	s.StoreValue("json-key", map[string]interface{}{"name": "test"})
	s.StoreValue("string-key", "plain text value")

	tests := []struct {
		name           string
		path           string
		acceptHeader   string
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "get all - accept json",
			path:           "/system/store/test-store",
			acceptHeader:   "application/json",
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
		{
			name:           "get all - accept anything",
			path:           "/system/store/test-store",
			acceptHeader:   "*/*",
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
		{
			name:           "get all - no accept header",
			path:           "/system/store/test-store",
			acceptHeader:   "",
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
		{
			name:           "get all - accept html",
			path:           "/system/store/test-store",
			acceptHeader:   "text/html",
			expectedStatus: http.StatusNotAcceptable,
		},
		{
			name:           "get json item - accept anything",
			path:           "/system/store/test-store/json-key",
			acceptHeader:   "*/*",
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
		},
		{
			name:           "get string item - accept anything",
			path:           "/system/store/test-store/string-key",
			acceptHeader:   "*/*",
			expectedStatus: http.StatusOK,
			expectedType:   "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, tt.path, nil)
			assert.NoError(t, err)

			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			rr := httptest.NewRecorder()
			HandleStoreRequest(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedType, rr.Header().Get("Content-Type"))
			}
		})
	}
}

func TestHandlePutStore_ContentTypes(t *testing.T) {
	// Initialize store provider
	store.InitStoreProvider()

	tests := []struct {
		name           string
		path           string
		contentType    string
		body           string
		expectedStatus int
		validateFunc   func(t *testing.T)
	}{
		{
			name:           "put text/plain",
			path:           "/system/store/test-store/text-key",
			contentType:    "text/plain",
			body:           "plain text value",
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T) {
				s := store.Open("test-store", nil)
				value, found := s.GetValue("text-key")
				assert.True(t, found)
				assert.Equal(t, "plain text value", value)
			},
		},
		{
			name:           "put application/json",
			path:           "/system/store/test-store/json-key",
			contentType:    "application/json",
			body:           `{"name":"test"}`,
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T) {
				s := store.Open("test-store", nil)
				value, found := s.GetValue("json-key")
				assert.True(t, found)
				assert.Equal(t, `{"name":"test"}`, value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.DeleteStore("test-store")

			req, err := http.NewRequest(http.MethodPut, tt.path, bytes.NewBufferString(tt.body))
			assert.NoError(t, err)

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			rr := httptest.NewRecorder()
			HandleStoreRequest(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.validateFunc != nil {
				tt.validateFunc(t)
			}
		})
	}
}
