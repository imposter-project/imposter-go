package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate_Name(t *testing.T) {
	tests := []struct {
		property string
	}{
		{"firstName"},
		{"lastName"},
		{"fullName"},
		{"name"},
		{"prefix"},
		{"suffix"},
		{"username"},
		{"title"},
	}
	for _, tt := range tests {
		t.Run(tt.property, func(t *testing.T) {
			val, ok := Generate("Name", tt.property)
			assert.True(t, ok, "expected Name.%s to be found", tt.property)
			assert.NotEmpty(t, val, "expected Name.%s to produce a value", tt.property)
		})
	}
}

func TestGenerate_Internet(t *testing.T) {
	tests := []struct {
		property string
		check    func(string) bool
	}{
		{"emailAddress", func(s string) bool { return strings.Contains(s, "@") }},
		{"email", func(s string) bool { return strings.Contains(s, "@") }},
		{"url", func(s string) bool { return strings.HasPrefix(s, "http") }},
		{"domainName", func(s string) bool { return strings.Contains(s, ".") }},
		{"ipV4Address", func(s string) bool { return strings.Contains(s, ".") }},
		{"ipV6Address", func(s string) bool { return strings.Contains(s, ":") }},
		{"password", func(s string) bool { return len(s) > 0 }},
		{"userAgent", func(s string) bool { return len(s) > 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.property, func(t *testing.T) {
			val, ok := Generate("Internet", tt.property)
			assert.True(t, ok, "expected Internet.%s to be found", tt.property)
			assert.True(t, tt.check(val), "expected Internet.%s value '%s' to pass check", tt.property, val)
		})
	}
}

func TestGenerate_Address(t *testing.T) {
	properties := []string{"streetAddress", "city", "state", "stateAbbr", "country", "countryCode", "zipCode", "latitude", "longitude", "fullAddress"}
	for _, prop := range properties {
		t.Run(prop, func(t *testing.T) {
			val, ok := Generate("Address", prop)
			assert.True(t, ok, "expected Address.%s to be found", prop)
			assert.NotEmpty(t, val, "expected Address.%s to produce a value", prop)
		})
	}
}

func TestGenerate_PhoneNumber(t *testing.T) {
	val, ok := Generate("PhoneNumber", "phoneNumber")
	assert.True(t, ok)
	assert.NotEmpty(t, val)
}

func TestGenerate_Company(t *testing.T) {
	properties := []string{"name", "industry", "buzzword", "catchPhrase", "bs"}
	for _, prop := range properties {
		t.Run(prop, func(t *testing.T) {
			val, ok := Generate("Company", prop)
			assert.True(t, ok, "expected Company.%s to be found", prop)
			assert.NotEmpty(t, val)
		})
	}
}

func TestGenerate_Lorem(t *testing.T) {
	properties := []string{"word", "sentence", "paragraph", "characters"}
	for _, prop := range properties {
		t.Run(prop, func(t *testing.T) {
			val, ok := Generate("Lorem", prop)
			assert.True(t, ok, "expected Lorem.%s to be found", prop)
			assert.NotEmpty(t, val)
		})
	}
}

func TestGenerate_Color(t *testing.T) {
	val, ok := Generate("Color", "name")
	assert.True(t, ok)
	assert.NotEmpty(t, val)

	val, ok = Generate("Color", "hex")
	assert.True(t, ok)
	assert.True(t, strings.HasPrefix(val, "#"))
}

func TestGenerate_Number(t *testing.T) {
	val, ok := Generate("Number", "digit")
	assert.True(t, ok)
	assert.NotEmpty(t, val)

	val, ok = Generate("Number", "randomNumber")
	assert.True(t, ok)
	assert.NotEmpty(t, val)
}

func TestGenerate_Bool(t *testing.T) {
	val, ok := Generate("Bool", "bool")
	assert.True(t, ok)
	assert.True(t, val == "true" || val == "false")
}

func TestGenerate_Finance(t *testing.T) {
	val, ok := Generate("Finance", "creditCardNumber")
	assert.True(t, ok)
	assert.NotEmpty(t, val)
}

func TestGenerate_Date(t *testing.T) {
	properties := []string{"past", "future", "birthday"}
	for _, prop := range properties {
		t.Run(prop, func(t *testing.T) {
			val, ok := Generate("Date", prop)
			assert.True(t, ok, "expected Date.%s to be found", prop)
			assert.NotEmpty(t, val)
		})
	}
}

func TestGenerate_CaseInsensitive(t *testing.T) {
	val, ok := Generate("name", "firstname")
	assert.True(t, ok)
	assert.NotEmpty(t, val)

	val, ok = Generate("NAME", "LASTNAME")
	assert.True(t, ok)
	assert.NotEmpty(t, val)
}

func TestGenerate_UnknownCategory(t *testing.T) {
	val, ok := Generate("Unknown", "thing")
	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestGenerate_UnknownProperty(t *testing.T) {
	val, ok := Generate("Name", "unknownProp")
	assert.False(t, ok)
	assert.Empty(t, val)
}
