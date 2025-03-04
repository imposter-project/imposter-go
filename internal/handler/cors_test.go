package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestHandleCORS(t *testing.T) {
	tests := []struct {
		name        string
		corsConfig  *config.CorsConfig
		method      string
		origin      string
		headers     map[string]string
		wantHandled bool
		wantStatus  int
		wantHeaders map[string]string
	}{
		{
			name: "preflight request with specific origin allowed",
			corsConfig: &config.CorsConfig{
				AllowOrigins:     []string{"http://localhost:3000"},
				AllowMethods:     []string{"GET", "POST"},
				AllowHeaders:     []string{"Content-Type", "Authorization"},
				MaxAge:           3600,
				AllowCredentials: true,
			},
			method: "OPTIONS",
			origin: "http://localhost:3000",
			headers: map[string]string{
				"Access-Control-Request-Method":  "POST",
				"Access-Control-Request-Headers": "Content-Type",
			},
			wantHandled: true,
			wantStatus:  http.StatusNoContent,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "http://localhost:3000",
				"Access-Control-Allow-Methods":     "GET, POST",
				"Access-Control-Allow-Headers":     "Content-Type, Authorization",
				"Access-Control-Allow-Credentials": "true",
				"Access-Control-Max-Age":           "3600",
				"Vary":                             "Origin",
			},
		},
		{
			name: "preflight request with 'all' as string - echoes back origin",
			corsConfig: &config.CorsConfig{
				AllowOrigins: "all",
				AllowMethods: []string{"GET", "POST", "PUT"},
				AllowHeaders: []string{"Content-Type", "Authorization"},
			},
			method: "OPTIONS",
			origin: "https://example.org",
			headers: map[string]string{
				"Access-Control-Request-Method": "POST",
			},
			wantHandled: true,
			wantStatus:  http.StatusNoContent,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "https://example.org",
				"Access-Control-Allow-Methods": "GET, POST, PUT",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
				"Vary":                         "Origin",
			},
		},
		{
			name: "regular request with 'all' as string - echoes back origin",
			corsConfig: &config.CorsConfig{
				AllowOrigins: "all",
			},
			method:      "GET",
			origin:      "https://app.example.org",
			wantHandled: false,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "https://app.example.org",
				"Vary":                        "Origin",
			},
		},
		{
			name: "preflight request with wildcard origin",
			corsConfig: &config.CorsConfig{
				AllowOrigins: "*",
				AllowMethods: []string{"GET", "POST", "PUT"},
			},
			method: "OPTIONS",
			origin: "http://example.com",
			headers: map[string]string{
				"Access-Control-Request-Method": "PUT",
			},
			wantHandled: true,
			wantStatus:  http.StatusNoContent,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PUT",
			},
		},
		{
			name: "preflight request with disallowed origin",
			corsConfig: &config.CorsConfig{
				AllowOrigins: []string{"http://localhost:3000"},
				AllowMethods: []string{"GET", "POST"},
			},
			method: "OPTIONS",
			origin: "http://evil.com",
			headers: map[string]string{
				"Access-Control-Request-Method": "POST",
			},
			wantHandled: true,
			wantStatus:  http.StatusNoContent,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Methods": "GET, POST",
			},
		},
		{
			name: "non-preflight request",
			corsConfig: &config.CorsConfig{
				AllowOrigins: []string{"http://localhost:3000"},
				AllowMethods: []string{"GET"},
			},
			method:      "GET",
			origin:      "http://localhost:3000",
			wantHandled: false,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "http://localhost:3000",
				"Vary":                        "Origin",
			},
		},
		{
			name: "preflight request without origin header returns 400",
			corsConfig: &config.CorsConfig{
				AllowOrigins: []string{"http://localhost:3000"},
				AllowMethods: []string{"GET", "POST"},
			},
			method: "OPTIONS",
			headers: map[string]string{
				"Access-Control-Request-Method": "POST",
			},
			wantHandled: true,
			wantStatus:  http.StatusBadRequest,
			wantHeaders: map[string]string{},
		},
		{
			name: "regular request without origin header should not set CORS headers",
			corsConfig: &config.CorsConfig{
				AllowOrigins: []string{"http://localhost:3000"},
				AllowMethods: []string{"GET", "POST"},
			},
			method:      "GET",
			wantHandled: false,
			wantHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/test", nil)

			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			handled := handleCORS(w, req, tt.corsConfig)
			assert.Equal(t, tt.wantHandled, handled, "handleCORS handled status")

			if tt.wantHandled {
				assert.Equal(t, tt.wantStatus, w.Code, "response status code")
			}

			for k, v := range tt.wantHeaders {
				assert.Equal(t, v, w.Header().Get(k), "response header %s", k)
			}
		})
	}
}

func TestAddCORSHeaders(t *testing.T) {
	tests := []struct {
		name        string
		corsConfig  *config.CorsConfig
		origin      string
		wantHeaders map[string]string
	}{
		{
			name: "add headers with specific origin",
			corsConfig: &config.CorsConfig{
				AllowOrigins:     []string{"http://localhost:3000"},
				AllowCredentials: true,
			},
			origin: "http://localhost:3000",
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "http://localhost:3000",
				"Access-Control-Allow-Credentials": "true",
				"Vary":                             "Origin",
			},
		},
		{
			name: "add headers with 'all' as string",
			corsConfig: &config.CorsConfig{
				AllowOrigins: "all",
			},
			origin: "http://example.com",
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "http://example.com",
				"Vary":                        "Origin",
			},
		},
		{
			name: "add headers with wildcard origin",
			corsConfig: &config.CorsConfig{
				AllowOrigins: "*",
			},
			origin: "http://example.com",
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
		},
		{
			name: "add headers with multiple allowed origins",
			corsConfig: &config.CorsConfig{
				AllowOrigins: []string{"http://localhost:3000", "http://example.com"},
			},
			origin: "http://example.com",
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin": "http://example.com",
				"Vary":                        "Origin",
			},
		},
		{
			name: "add headers with disallowed origin",
			corsConfig: &config.CorsConfig{
				AllowOrigins: []string{"http://localhost:3000"},
			},
			origin:      "http://evil.com",
			wantHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)

			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			addCORSHeaders(w, req, tt.corsConfig)

			for k, v := range tt.wantHeaders {
				assert.Equal(t, v, w.Header().Get(k), "response header %s", k)
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		str      string
		expected bool
	}{
		{
			name:     "string exists in slice",
			slice:    []string{"one", "two", "three"},
			str:      "two",
			expected: true,
		},
		{
			name:     "string does not exist in slice",
			slice:    []string{"one", "two", "three"},
			str:      "four",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			str:      "one",
			expected: false,
		},
		{
			name:     "nil slice",
			slice:    nil,
			str:      "one",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsString(tt.slice, tt.str)
			assert.Equal(t, tt.expected, result)
		})
	}
}
