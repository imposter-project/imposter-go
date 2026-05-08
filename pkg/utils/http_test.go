package utils

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		name         string
		requestPath  string
		resourcePath string
		expected     map[string]string
	}{
		{
			name:         "Empty paths",
			requestPath:  "",
			resourcePath: "",
			expected:     map[string]string{},
		},
		{
			name:         "No parameters",
			requestPath:  "/pets/dogs",
			resourcePath: "/pets/dogs",
			expected:     map[string]string{},
		},
		{
			name:         "Single parameter",
			requestPath:  "/pets/123",
			resourcePath: "/pets/{id}",
			expected: map[string]string{
				"id": "123",
			},
		},
		{
			name:         "Multiple parameters",
			requestPath:  "/pets/123/photos/456",
			resourcePath: "/pets/{petId}/photos/{photoId}",
			expected: map[string]string{
				"petId":   "123",
				"photoId": "456",
			},
		},
		{
			name:         "Parameters with trailing slash",
			requestPath:  "/pets/123/",
			resourcePath: "/pets/{id}/",
			expected: map[string]string{
				"id": "123",
			},
		},
		{
			name:         "Mixed static and parameter segments",
			requestPath:  "/api/v1/pets/123/photos/456",
			resourcePath: "/api/v1/pets/{petId}/photos/{photoId}",
			expected: map[string]string{
				"petId":   "123",
				"photoId": "456",
			},
		},
		{
			name:         "Parameters with special characters",
			requestPath:  "/pets/abc-123/photos/def_456",
			resourcePath: "/pets/{petId}/photos/{photoId}",
			expected: map[string]string{
				"petId":   "abc-123",
				"photoId": "def_456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPathParams(tt.requestPath, tt.resourcePath)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ExtractPathParams() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func newMultipartRequest(t *testing.T, fields map[string]string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("failed to write field %q: %v", k, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, "/", &buf)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestGetFormParams_URLEncoded(t *testing.T) {
	body := "key1=value1&key2=value2"
	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got := GetFormParams(req)
	want := map[string]string{"key1": "value1", "key2": "value2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetFormParams() = %v, want %v", got, want)
	}
}

func TestGetFormParams_Multipart(t *testing.T) {
	req := newMultipartRequest(t, map[string]string{
		"name":  "alice",
		"email": "alice@example.com",
	})

	got := GetFormParams(req)
	want := map[string]string{"name": "alice", "email": "alice@example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetFormParams() = %v, want %v", got, want)
	}
}

func TestGetFormParams_NoBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	got := GetFormParams(req)
	if len(got) != 0 {
		t.Errorf("GetFormParams() = %v, want empty map", got)
	}
}

func TestGetFormValue_URLEncoded(t *testing.T) {
	body := "field=form-data"
	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if got := GetFormValue(req, "field"); got != "form-data" {
		t.Errorf("GetFormValue() = %q, want %q", got, "form-data")
	}
}

func TestGetFormValue_Multipart(t *testing.T) {
	req := newMultipartRequest(t, map[string]string{"field": "form-data"})

	if got := GetFormValue(req, "field"); got != "form-data" {
		t.Errorf("GetFormValue() = %q, want %q", got, "form-data")
	}
}
