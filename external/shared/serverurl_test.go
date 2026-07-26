package shared

import "testing"

func TestServerBasePath(t *testing.T) {
	tests := []struct {
		name      string
		serverURL string
		expected  string
	}{
		{name: "no path", serverURL: "http://localhost:8080", expected: ""},
		{name: "root path only", serverURL: "http://localhost:8080/", expected: ""},
		{name: "single segment", serverURL: "http://localhost:8080/myapp", expected: "/myapp"},
		{name: "single segment with trailing slash", serverURL: "http://localhost:8080/myapp/", expected: "/myapp"},
		{name: "multiple segments", serverURL: "https://example.com/foo/bar", expected: "/foo/bar"},
		{name: "empty string", serverURL: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ServerBasePath(tt.serverURL); got != tt.expected {
				t.Errorf("ServerBasePath(%q) = %q, want %q", tt.serverURL, got, tt.expected)
			}
		})
	}
}
