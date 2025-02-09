package steps

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestExecuteRemoteStep(t *testing.T) {
	store.InitStoreProvider()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status": "created", "code": "201"}`))
	}))
	defer server.Close()

	imposterConfig := &config.ImposterConfig{
		ServerPort: "8080",
	}

	tests := []struct {
		name        string
		step        config.Step
		setupExch   func() *exchange.Exchange
		validate    func(t *testing.T, store *store.Store)
		expectError bool
	}{
		{
			name: "basic GET request",
			step: config.Step{
				Type:   config.RemoteStepType,
				Method: "GET",
				URL:    server.URL + "/test",
			},
			setupExch: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/test", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte{},
					},
					RequestStore: &store.Store{},
				}
			},
		},
		{
			name: "POST request with body and headers",
			step: config.Step{
				Type:   config.RemoteStepType,
				Method: "POST",
				URL:    server.URL + "/create",
				Body:   `{"data": "test data"}`,
				Headers: map[string]string{
					"Content-Type": "application/json",
					"X-Request-ID": "123",
				},
			},
			setupExch: func() *exchange.Exchange {
				req, _ := http.NewRequest("POST", "/test", strings.NewReader("test data"))
				req.Header.Set("X-Request-ID", "123")
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte("test data"),
					},
					RequestStore: &store.Store{},
				}
			},
		},
		{
			name: "capture response data",
			step: config.Step{
				Type:   config.RemoteStepType,
				Method: "POST",
				URL:    server.URL + "/create",
				Body:   `{"name": "test"}`,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Capture: map[string]config.Capture{
					"status": {
						Store: "request",
						CaptureConfig: config.CaptureConfig{
							Expression: "${context.response.body:$.status}",
						},
					},
					"code": {
						Store: "request",
						CaptureConfig: config.CaptureConfig{
							Expression: "${context.response.body:$.code}",
						},
					},
				},
			},
			setupExch: func() *exchange.Exchange {
				req, _ := http.NewRequest("POST", "/create", strings.NewReader(`{"name": "test"}`))
				s := store.Store{}
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte(`{"name": "test"}`),
					},
					RequestStore: &s,
				}
			},
			validate: func(t *testing.T, s *store.Store) {
				status, exists := (*s)["status"]
				assert.True(t, exists)
				assert.Equal(t, "created", status)
				code, exists := (*s)["code"]
				assert.True(t, exists)
				assert.Equal(t, "201", code)
			},
		},
		{
			name: "invalid URL",
			step: config.Step{
				Type:   config.RemoteStepType,
				Method: "GET",
				URL:    "invalid-url",
			},
			setupExch: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/test", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte{},
					},
					RequestStore: &store.Store{},
				}
			},
			expectError: true,
		},
		{
			name: "server error",
			step: config.Step{
				Type:   config.RemoteStepType,
				Method: "GET",
				URL:    server.URL + "/error",
			},
			setupExch: func() *exchange.Exchange {
				req, _ := http.NewRequest("GET", "/test", nil)
				return &exchange.Exchange{
					Request: &exchange.RequestContext{
						Request: req,
						Body:    []byte{},
					},
					RequestStore: &store.Store{},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exch := tt.setupExch()
			err := executeRemoteStep(&tt.step, exch, imposterConfig)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t, exch.RequestStore)
			}
		})
	}
}
