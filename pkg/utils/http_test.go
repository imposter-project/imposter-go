package utils

import (
	"reflect"
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
