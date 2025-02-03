package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetFirstItemFromMap(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]string
		expectedKey   string
		expectedValue string
	}{
		{
			name:          "Empty map",
			input:         map[string]string{},
			expectedKey:   "",
			expectedValue: "",
		},
		{
			name: "Single item",
			input: map[string]string{
				"key1": "value1",
			},
			expectedKey:   "key1",
			expectedValue: "value1",
		},
		{
			name: "Multiple items - alphabetical order",
			input: map[string]string{
				"key3": "value3",
				"key1": "value1",
				"key2": "value2",
			},
			expectedKey:   "key1",
			expectedValue: "value1",
		},
		{
			name: "Empty strings",
			input: map[string]string{
				"":    "empty-key",
				"key": "",
			},
			expectedKey:   "",
			expectedValue: "empty-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value := GetFirstItemFromMap(tt.input)
			require.Equal(t, tt.expectedKey, key, "key mismatch")
			require.Equal(t, tt.expectedValue, value, "value mismatch")
		})
	}
}
