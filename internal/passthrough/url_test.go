package passthrough

import "testing"

func TestJoinURL(t *testing.T) {
	tests := []struct {
		name        string
		base        string
		requestPath string
		rawQuery    string
		want        string
		wantErr     bool
	}{
		{name: "base path joined with request path", base: "http://api/v1", requestPath: "/users", want: "http://api/v1/users"},
		{name: "trailing slash on base normalised", base: "http://api/v1/", requestPath: "/users", want: "http://api/v1/users"},
		{name: "no base path", base: "http://api", requestPath: "/v1/users", want: "http://api/v1/users"},
		{name: "root request path preserves trailing slash", base: "http://api/v1", requestPath: "/", want: "http://api/v1/"},
		{name: "query string forwarded verbatim", base: "http://api", requestPath: "/users", rawQuery: "a=1&b=two", want: "http://api/users?a=1&b=two"},
		{name: "query not re-encoded", base: "http://api", requestPath: "/search", rawQuery: "q=a+b%20c", want: "http://api/search?q=a+b%20c"},
		{name: "port preserved", base: "http://api:8080/base", requestPath: "/users", want: "http://api:8080/base/users"},
		{name: "empty request path keeps base", base: "http://api/v1", requestPath: "", want: "http://api/v1"},
		{name: "missing scheme errors", base: "api/v1", requestPath: "/users", wantErr: true},
		{name: "missing host errors", base: "http://", requestPath: "/users", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := JoinURL(tt.base, tt.requestPath, tt.rawQuery)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("JoinURL(%q, %q, %q) = %q, want %q", tt.base, tt.requestPath, tt.rawQuery, got, tt.want)
			}
		})
	}
}
