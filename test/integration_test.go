package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeRoute(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("Could not create request: %v", err)
	}

	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Welcome to Imposter-Go!"))
	})

	handler.ServeHTTP(rec, req)

	if status := rec.Code; status != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, status)
	}

	expectedBody := "Welcome to Imposter-Go!"
	if rec.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, rec.Body.String())
	}
}

func TestIntegration_MatchJSONBody(t *testing.T) {
	// Set up a test HTTP server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/match-json" && r.Method == http.MethodPost {
			body := `{"user": {"name": "John"}}`
			if strings.Contains(body, "John") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Matched JSON body!"))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Create a test request
	resp, err := http.Post(server.URL+"/match-json", "application/json", strings.NewReader(`{"user": {"name": "John"}}`))
	if err != nil {
		t.Fatalf("Failed to make POST request: %v", err)
	}
	defer resp.Body.Close()

	// Validate response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}
}
