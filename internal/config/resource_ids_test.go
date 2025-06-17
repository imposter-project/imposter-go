package config

import (
	"testing"
)

func TestPreCalculateResourceIDs(t *testing.T) {
	// Create test configurations
	configs := []Config{
		{
			Plugin: "rest",
			Resources: []Resource{
				{
					BaseResource: BaseResource{
						RequestMatcher: RequestMatcher{
							Method: "GET",
							Path:   "/api/users",
						},
					},
				},
				{
					BaseResource: BaseResource{
						RequestMatcher: RequestMatcher{
							Method: "GET",
							Path:   "/api/users",
							RequestHeaders: map[string]MatcherUnmarshaler{
								"Authorization": {},
							},
						},
					},
				},
			},
			Interceptors: []Interceptor{
				{
					BaseResource: BaseResource{
						RequestMatcher: RequestMatcher{
							Method: "POST",
							Path:   "/api/validate",
						},
					},
				},
			},
		},
		{
			Plugin: "soap",
			Resources: []Resource{
				{
					BaseResource: BaseResource{
						RequestMatcher: RequestMatcher{
							Operation: "getUserDetails",
						},
					},
				},
				{
					BaseResource: BaseResource{
						RequestMatcher: RequestMatcher{
							Operation:  "getUserDetails",
							SOAPAction: "getUserAction",
						},
					},
				},
			},
			Interceptors: []Interceptor{
				{
					BaseResource: BaseResource{
						RequestMatcher: RequestMatcher{
							Operation: "validateUser",
						},
					},
				},
			},
		},
	}

	// Pre-calculate IDs
	PreCalculateResourceIDs(configs)

	// Verify REST resources have correct IDs
	t.Run("REST resources", func(t *testing.T) {
		resource1 := &configs[0].Resources[0]
		resource2 := &configs[0].Resources[1]

		// Both resources have the same method/path but different headers
		if resource1.ResourceID == "" {
			t.Error("Resource 1 resource ID not calculated")
		}
		if resource2.ResourceID == "" {
			t.Error("Resource 2 resource ID not calculated")
		}

		// IDs should be different due to different headers
		if resource1.ResourceID == resource2.ResourceID {
			t.Errorf("Expected different IDs for resources with different headers, got same: %s", resource1.ResourceID)
		}

		// Resource 1 should have simple ID (no extra criteria)
		expectedID1 := "GET:/api/users"
		if resource1.ResourceID != expectedID1 {
			t.Errorf("Expected ID1 to be %s, got %s", expectedID1, resource1.ResourceID)
		}

		// Resource 2 should have hash suffix (due to headers)
		if len(resource2.ResourceID) <= len(expectedID1) {
			t.Errorf("Expected ID2 to be longer than base ID due to hash, got %s", resource2.ResourceID)
		}

		t.Logf("Resource 1 ID: %s", resource1.ResourceID)
		t.Logf("Resource 2 ID: %s", resource2.ResourceID)
	})

	// Verify REST interceptor has correct ID
	t.Run("REST interceptor", func(t *testing.T) {
		interceptor := &configs[0].Interceptors[0]
		expectedID := "POST:/api/validate"

		if interceptor.ResourceID != expectedID {
			t.Errorf("Expected interceptor ID to be %s, got %s", expectedID, interceptor.ResourceID)
		}
	})

	// Verify SOAP resources have correct IDs
	t.Run("SOAP resources", func(t *testing.T) {
		resource1 := &configs[1].Resources[0]
		resource2 := &configs[1].Resources[1]

		if resource1.ResourceID == "" {
			t.Error("SOAP Resource 1 resource ID not calculated")
		}
		if resource2.ResourceID == "" {
			t.Error("SOAP Resource 2 resource ID not calculated")
		}

		// IDs should be different due to different SOAPAction
		if resource1.ResourceID == resource2.ResourceID {
			t.Errorf("Expected different IDs for SOAP resources with different criteria, got same: %s", resource1.ResourceID)
		}

		// Resource 1 should have simple ID
		expectedID1 := "*:getUserDetails" // No method specified, defaults to *
		if resource1.ResourceID != expectedID1 {
			t.Errorf("Expected SOAP ID1 to be %s, got %s", expectedID1, resource1.ResourceID)
		}

		t.Logf("SOAP Resource 1 ID: %s", resource1.ResourceID)
		t.Logf("SOAP Resource 2 ID: %s", resource2.ResourceID)
	})

	// Verify SOAP interceptor has correct ID
	t.Run("SOAP interceptor", func(t *testing.T) {
		interceptor := &configs[1].Interceptors[0]
		expectedID := "*:validateUser" // No method specified, defaults to *

		if interceptor.ResourceID != expectedID {
			t.Errorf("Expected SOAP interceptor ID to be %s, got %s", expectedID, interceptor.ResourceID)
		}
	})
}

func TestPreCalculateResourceIDs_EmptyConfigs(t *testing.T) {
	// Test with empty configs - should not panic
	configs := []Config{}
	PreCalculateResourceIDs(configs)

	// Test with configs with no resources/interceptors
	configs = []Config{
		{Plugin: "rest"},
		{Plugin: "soap"},
	}
	PreCalculateResourceIDs(configs)
}

func TestGenerateResourceKey(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		resName  string
		matcher  *RequestMatcher
		expected string
	}{
		{
			name:     "simple resource",
			method:   "GET",
			resName:  "/api/users",
			matcher:  nil,
			expected: "GET:/api/users",
		},
		{
			name:     "empty matcher",
			method:   "POST",
			resName:  "/api/data",
			matcher:  &RequestMatcher{},
			expected: "POST:/api/data",
		},
		{
			name:    "with headers",
			method:  "GET",
			resName: "/api/users",
			matcher: &RequestMatcher{
				RequestHeaders: map[string]MatcherUnmarshaler{
					"Authorization": {},
				},
			},
			expected: "GET:/api/users:b59e2e91", // hash will be deterministic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateResourceKey(tt.method, tt.resName, tt.matcher)
			if result != tt.expected {
				t.Errorf("GenerateResourceKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}
