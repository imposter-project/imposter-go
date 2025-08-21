package external

import (
	"bytes"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/external/shared"
	"github.com/imposter-project/imposter-go/internal/exchange"
)

func Test_convertToSingleValueHeaders(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   map[string]string
	}{
		{
			name:   "empty headers",
			header: http.Header{},
			want:   map[string]string{},
		},
		{
			name: "single value headers",
			header: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Bearer token123"},
				"User-Agent":    []string{"Mozilla/5.0"},
			},
			want: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token123",
				"User-Agent":    "Mozilla/5.0",
			},
		},
		{
			name: "multiple value headers - takes first value",
			header: http.Header{
				"Accept":          []string{"text/html", "application/json", "*/*"},
				"Accept-Encoding": []string{"gzip", "deflate", "br"},
				"Cache-Control":   []string{"no-cache", "no-store"},
			},
			want: map[string]string{
				"Accept":          "text/html",
				"Accept-Encoding": "gzip",
				"Cache-Control":   "no-cache",
			},
		},
		{
			name: "mixed single and multiple values",
			header: http.Header{
				"Content-Type":  []string{"application/xml"},
				"Accept":        []string{"application/json", "text/plain"},
				"Authorization": []string{"Basic dGVzdDp0ZXN0"},
				"X-Custom":      []string{"value1", "value2", "value3"},
			},
			want: map[string]string{
				"Content-Type":  "application/xml",
				"Accept":        "application/json",
				"Authorization": "Basic dGVzdDp0ZXN0",
				"X-Custom":      "value1",
			},
		},
		{
			name: "headers with empty values",
			header: http.Header{
				"Valid-Header": []string{"valid-value"},
				"Empty-Header": []string{},
			},
			want: map[string]string{
				"Valid-Header": "valid-value",
				// Empty-Header should not appear in result
			},
		},
		{
			name: "case sensitivity preserved",
			header: http.Header{
				"content-type":  []string{"text/html"},
				"Content-Type":  []string{"application/json"},
				"AUTHORIZATION": []string{"Bearer xyz"},
			},
			want: map[string]string{
				"content-type":  "text/html",
				"Content-Type":  "application/json",
				"AUTHORIZATION": "Bearer xyz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToSingleValueHeaders(tt.header)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertToSingleValueHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertToSingleValueHeaders_EmptyValueHandling(t *testing.T) {
	// Test specifically for empty slice handling
	header := http.Header{
		"Has-Value":   []string{"test"},
		"Empty-Slice": []string{},
		"Has-Empty":   []string{""},
	}

	result := convertToSingleValueHeaders(header)

	// Should have "Has-Value" with "test"
	if result["Has-Value"] != "test" {
		t.Errorf("Expected 'Has-Value' to be 'test', got '%s'", result["Has-Value"])
	}

	// Should NOT have "Empty-Slice" since it has no values
	if _, exists := result["Empty-Slice"]; exists {
		t.Errorf("Expected 'Empty-Slice' to not exist in result, but it does")
	}

	// Should have "Has-Empty" with empty string value
	if val, exists := result["Has-Empty"]; !exists {
		t.Errorf("Expected 'Has-Empty' to exist in result")
	} else if val != "" {
		t.Errorf("Expected 'Has-Empty' to be empty string, got '%s'", val)
	}

	// Result should have exactly 2 keys
	if len(result) != 2 {
		t.Errorf("Expected result to have 2 keys, got %d: %v", len(result), result)
	}
}

func TestConvertToExternalRequest(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func() *exchange.Exchange
		want     shared.HandlerRequest
	}{
		{
			name: "basic GET request",
			setupReq: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/api/test?param=value", nil)
				req.Header.Set("Authorization", "Bearer token123")
				req.Header.Set("Content-Type", "application/json")

				exch := &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte("test body"),
					},
				}
				return exch
			},
			want: shared.HandlerRequest{
				Method: "GET",
				Path:   "/api/test",
				Query: url.Values{
					"param": []string{"value"},
				},
				Headers: map[string]string{
					"Authorization": "Bearer token123",
					"Content-Type":  "application/json",
				},
				Body: []byte("test body"),
			},
		},
		{
			name: "POST request with multiple query params",
			setupReq: func() *exchange.Exchange {
				req, _ := http.NewRequest("POST", "/submit?user=alice&role=admin&role=user", strings.NewReader("form data"))
				req.Header.Set("User-Agent", "Test/1.0")
				req.Header.Add("Accept", "text/html")
				req.Header.Add("Accept", "application/json") // Multiple values

				exch := &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte("form data"),
					},
				}
				return exch
			},
			want: shared.HandlerRequest{
				Method: "POST",
				Path:   "/submit",
				Query: url.Values{
					"user": []string{"alice"},
					"role": []string{"admin", "user"},
				},
				Headers: map[string]string{
					"User-Agent": "Test/1.0",
					"Accept":     "text/html", // Should take first value
				},
				Body: []byte("form data"),
			},
		},
		{
			name: "request with no headers or body",
			setupReq: func() *exchange.Exchange {
				req, _ := http.NewRequest("DELETE", "/api/item/123", nil)

				exch := &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte{},
					},
				}
				return exch
			},
			want: shared.HandlerRequest{
				Method:  "DELETE",
				Path:    "/api/item/123",
				Query:   url.Values{},
				Headers: map[string]string{},
				Body:    []byte{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exch := tt.setupReq()
			got := ConvertToExternalRequest(exch)

			if got.Method != tt.want.Method {
				t.Errorf("ConvertToExternalRequest() Method = %v, want %v", got.Method, tt.want.Method)
			}
			if got.Path != tt.want.Path {
				t.Errorf("ConvertToExternalRequest() Path = %v, want %v", got.Path, tt.want.Path)
			}
			if !reflect.DeepEqual(got.Query, tt.want.Query) {
				t.Errorf("ConvertToExternalRequest() Query = %v, want %v", got.Query, tt.want.Query)
			}
			if !reflect.DeepEqual(got.Headers, tt.want.Headers) {
				t.Errorf("ConvertToExternalRequest() Headers = %v, want %v", got.Headers, tt.want.Headers)
			}
			if !bytes.Equal(got.Body, tt.want.Body) {
				t.Errorf("ConvertToExternalRequest() Body = %v, want %v", got.Body, tt.want.Body)
			}
		})
	}
}

func TestConvertFromExternalResponse(t *testing.T) {
	tests := []struct {
		name         string
		handlerResp  shared.HandlerResponse
		expectStatus int
		expectBody   []byte
		expectFile   string
	}{
		{
			name: "basic JSON response",
			handlerResp: shared.HandlerResponse{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Cache-Control": "no-cache",
				},
				Body: []byte(`{"message": "success"}`),
			},
			expectStatus: 200,
			expectBody:   []byte(`{"message": "success"}`),
			expectFile:   "",
		},
		{
			name: "file response",
			handlerResp: shared.HandlerResponse{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Disposition": "attachment; filename=test.pdf",
				},
				File:     "/path/to/file.pdf",
				FileName: "test.pdf",
			},
			expectStatus: 200,
			expectBody:   nil,
			expectFile:   "/path/to/file.pdf",
		},
		{
			name: "error response",
			handlerResp: shared.HandlerResponse{
				StatusCode: 404,
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
				Body: []byte("Not Found"),
			},
			expectStatus: 404,
			expectBody:   []byte("Not Found"),
			expectFile:   "",
		},
		{
			name: "empty response",
			handlerResp: shared.HandlerResponse{
				StatusCode: 204,
				Headers:    map[string]string{},
			},
			expectStatus: 204,
			expectBody:   nil,
			expectFile:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock exchange with response state
			exch := &exchange.Exchange{
				ResponseState: &exchange.ResponseState{
					Handled: false,
					Headers: make(map[string]string),
				},
			}

			// Call the function under test
			ConvertFromExternalResponse(exch, &tt.handlerResp)

			// Verify the response state was updated correctly
			rs := exch.ResponseState

			if !rs.Handled {
				t.Error("Expected ResponseState.Handled to be true")
			}

			if rs.StatusCode != tt.expectStatus {
				t.Errorf("Expected StatusCode %d, got %d", tt.expectStatus, rs.StatusCode)
			}

			if !bytes.Equal(rs.Body, tt.expectBody) {
				t.Errorf("Expected Body %v, got %v", tt.expectBody, rs.Body)
			}

			if rs.File != tt.expectFile {
				t.Errorf("Expected File %q, got %q", tt.expectFile, rs.File)
			}
		})
	}
}

func TestConvertToExternalRequest_NilSafety(t *testing.T) {
	// Test with minimal exchange to ensure no panics
	req, _ := http.NewRequest("GET", "/", nil)
	exch := &exchange.Exchange{
		Request: &exchange.RequestContext{
			Request: req,
			Body:    nil,
		},
	}

	result := ConvertToExternalRequest(exch)

	// Should not panic and should return valid result
	if result.Method != "GET" {
		t.Errorf("Expected Method GET, got %s", result.Method)
	}
	if result.Path != "/" {
		t.Errorf("Expected Path /, got %s", result.Path)
	}
	if result.Headers == nil {
		t.Error("Expected Headers to be non-nil map")
	}
	if result.Query == nil {
		t.Error("Expected Query to be non-nil")
	}
}
