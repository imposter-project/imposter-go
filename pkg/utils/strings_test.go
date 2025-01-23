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

func TestRemoveEmptyStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "No empty strings",
			input:    []string{"first", "second", "third"},
			expected: []string{"first", "second", "third"},
		},
		{
			name:     "Whitespace strings",
			input:    []string{" ", "\t", "\n", "\r", " \t\n\r "},
			expected: []string{},
		},
		{
			name:     "Strings with whitespace",
			input:    []string{" first ", "\tsecond\t", "\nthird\n"},
			expected: []string{"first", "second", "third"},
		},
		{
			name:     "Mixed content",
			input:    []string{"first", "", " second ", "\t", "third\n", " ", "fourth"},
			expected: []string{"first", "second", "third", "fourth"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveEmptyStrings(tt.input)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("RemoveEmptyStrings() returned slice of length %v, want %v", len(result), len(tt.expected))
				return
			}

			// Check contents
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("RemoveEmptyStrings()[%d] = %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}
