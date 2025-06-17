package remote

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/imposter-project/imposter-go/internal/capture"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/template"
)

// ExecuteRemoteStep executes a remote HTTP request step
func ExecuteRemoteStep(step *config.Step, exch *exchange.Exchange, imposterConfig *config.ImposterConfig) error {
	// Process templates in URL, headers, and body
	url := template.ProcessTemplate(step.URL, exch, imposterConfig, &config.RequestMatcher{})
	body := template.ProcessTemplate(step.Body, exch, imposterConfig, &config.RequestMatcher{})

	// Create request
	req, err := http.NewRequest(step.Method, url, bytes.NewReader([]byte(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Process and set headers
	for key, value := range step.Headers {
		processedValue := template.ProcessTemplate(value, exch, imposterConfig, &config.RequestMatcher{})
		req.Header.Set(key, processedValue)
	}

	// Set Content-Type if not specified and body is present
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Create response context
	// TODO: create a new exchange instead of updating the existing one
	exch.Response = &exchange.ResponseContext{
		Response: resp,
		Body:     respBody,
	}

	// Process captures using the common capture logic
	if len(step.Capture) > 0 {
		capture.CaptureRequestData(imposterConfig, &config.RequestMatcher{}, step.Capture, exch)
	}

	return nil
}
