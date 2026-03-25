package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateForPropertyName(t *testing.T) {
	tests := []struct {
		propertyName string
		check        func(string) bool
	}{
		{"firstName", func(s string) bool { return len(s) > 0 }},
		{"first_name", func(s string) bool { return len(s) > 0 }},
		{"lastName", func(s string) bool { return len(s) > 0 }},
		{"email", func(s string) bool { return strings.Contains(s, "@") }},
		{"emailAddress", func(s string) bool { return strings.Contains(s, "@") }},
		{"city", func(s string) bool { return len(s) > 0 }},
		{"country", func(s string) bool { return len(s) > 0 }},
		{"phone", func(s string) bool { return len(s) > 0 }},
		{"company", func(s string) bool { return len(s) > 0 }},
		{"url", func(s string) bool { return strings.HasPrefix(s, "http") }},
		{"username", func(s string) bool { return len(s) > 0 }},
		{"description", func(s string) bool { return len(s) > 0 }},
		{"color", func(s string) bool { return len(s) > 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.propertyName, func(t *testing.T) {
			val, ok := GenerateForPropertyName(tt.propertyName)
			assert.True(t, ok, "expected property name '%s' to be found", tt.propertyName)
			assert.True(t, tt.check(val), "expected value '%s' for property '%s' to pass check", val, tt.propertyName)
		})
	}
}

func TestGenerateForPropertyName_Unknown(t *testing.T) {
	val, ok := GenerateForPropertyName("unknownField")
	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestGenerateForFormat(t *testing.T) {
	tests := []struct {
		format string
		check  func(string) bool
	}{
		{"email", func(s string) bool { return strings.Contains(s, "@") }},
		{"uri", func(s string) bool { return strings.HasPrefix(s, "http") }},
		{"hostname", func(s string) bool { return strings.Contains(s, ".") }},
		{"ipv4", func(s string) bool { return strings.Contains(s, ".") }},
		{"ipv6", func(s string) bool { return strings.Contains(s, ":") }},
		{"password", func(s string) bool { return len(s) > 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			val, ok := GenerateForFormat(tt.format)
			assert.True(t, ok, "expected format '%s' to be found", tt.format)
			assert.True(t, tt.check(val), "expected value '%s' for format '%s' to pass check", val, tt.format)
		})
	}
}

func TestGenerateForFormat_Unknown(t *testing.T) {
	val, ok := GenerateForFormat("unknownFormat")
	assert.False(t, ok)
	assert.Empty(t, val)
}
