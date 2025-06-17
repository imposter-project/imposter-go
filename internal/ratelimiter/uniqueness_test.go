package ratelimiter

import (
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
)

// TestResourceKeyUniqueness verifies the specific issue mentioned in the GitHub issue:
// Resources with same path/method but different matching criteria get different keys
func TestResourceKeyUniqueness(t *testing.T) {
	// Test case from the issue: two resources with same path/method but different headers
	resource1 := &config.RequestMatcher{
		Method: "GET",
		Path:   "/foo",
		RequestHeaders: map[string]config.MatcherUnmarshaler{
			"bar": {},
		},
	}

	resource2 := &config.RequestMatcher{
		Method: "GET",
		Path:   "/foo",
		RequestHeaders: map[string]config.MatcherUnmarshaler{
			"qux": {},
		},
	}

	// Generate keys for both resources
	key1 := config.GenerateResourceKey("GET", "/foo", resource1)
	key2 := config.GenerateResourceKey("GET", "/foo", resource2)

	// Verify they have different keys
	if key1 == key2 {
		t.Errorf("Expected different keys for resources with different headers, got same key: %s", key1)
	}

	// Verify both keys start with the same base
	expectedBase := "GET:/foo"
	if !hasPrefix(key1, expectedBase) {
		t.Errorf("Expected key1 to start with %s, got: %s", expectedBase, key1)
	}
	if !hasPrefix(key2, expectedBase) {
		t.Errorf("Expected key2 to start with %s, got: %s", expectedBase, key2)
	}

	// Verify the keys have the correct format (method:name:hash)
	if countColons(key1) != 2 {
		t.Errorf("Expected key1 to have format 'method:name:hash', got: %s", key1)
	}
	if countColons(key2) != 2 {
		t.Errorf("Expected key2 to have format 'method:name:hash', got: %s", key2)
	}

	t.Logf("Generated unique keys: %s vs %s", key1, key2)
}

// TestMultiConfigScenario tests resources from multiple config files
func TestMultiConfigScenario(t *testing.T) {
	// Simulate resources from different config files with same method/path
	// but different matching criteria
	scenarios := []struct {
		name    string
		matcher *config.RequestMatcher
	}{
		{
			name: "config1_with_auth_header",
			matcher: &config.RequestMatcher{
				RequestHeaders: map[string]config.MatcherUnmarshaler{
					"Authorization": {},
				},
			},
		},
		{
			name: "config2_with_api_key",
			matcher: &config.RequestMatcher{
				RequestHeaders: map[string]config.MatcherUnmarshaler{
					"X-API-Key": {},
				},
			},
		},
		{
			name: "config3_with_query_param",
			matcher: &config.RequestMatcher{
				QueryParams: map[string]config.MatcherUnmarshaler{
					"version": {},
				},
			},
		},
		{
			name: "config4_with_form_param",
			matcher: &config.RequestMatcher{
				FormParams: map[string]config.MatcherUnmarshaler{
					"action": {},
				},
			},
		},
	}

	keys := make(map[string]string)

	// Generate keys for all scenarios
	for _, scenario := range scenarios {
		key := config.GenerateResourceKey("POST", "/api/data", scenario.matcher)
		keys[scenario.name] = key
		t.Logf("Scenario %s: %s", scenario.name, key)
	}

	// Verify all keys are unique
	seenKeys := make(map[string]string)
	for scenarioName, key := range keys {
		if existingScenario, exists := seenKeys[key]; exists {
			t.Errorf("Duplicate key found: %s used by both %s and %s", key, existingScenario, scenarioName)
		}
		seenKeys[key] = scenarioName
	}

	// Verify all keys have the same base
	expectedBase := "POST:/api/data"
	for scenarioName, key := range keys {
		if !hasPrefix(key, expectedBase) {
			t.Errorf("Key for %s should start with %s, got: %s", scenarioName, expectedBase, key)
		}
	}
}

// Helper functions
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func countColons(s string) int {
	count := 0
	for _, c := range s {
		if c == ':' {
			count++
		}
	}
	return count
}
