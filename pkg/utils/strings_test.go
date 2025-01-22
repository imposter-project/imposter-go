package utils

import "testing"

func TestStringSliceContainsElement(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		element  string
		expected bool
	}{
		{
			name:     "Empty slice",
			slice:    []string{},
			element:  "test",
			expected: false,
		},
		{
			name:     "Element present",
			slice:    []string{"first", "second", "third"},
			element:  "first",
			expected: true,
		},
		{
			name:     "Element not present",
			slice:    []string{"first", "second", "third"},
			element:  "fourth",
			expected: false,
		},
		{
			name:     "Empty element search",
			slice:    []string{"test", "other"},
			element:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringSliceContainsElement(&tt.slice, tt.element)
			if result != tt.expected {
				t.Errorf("StringSliceContainsElement() = %v, want %v", result, tt.expected)
			}
		})
	}
}
