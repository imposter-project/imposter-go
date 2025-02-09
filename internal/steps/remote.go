package steps

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

// executeRemoteStep executes a remote HTTP request step
func executeRemoteStep(step *config.Step, exch *exchange.Exchange, imposterConfig *config.ImposterConfig) error {
	// Process templates in URL, headers, and body
	url := template.ProcessTemplateWithContext(step.URL, exch, nil)
	body := template.ProcessTemplateWithContext(step.Body, exch, nil)

	// Create request
	req, err := http.NewRequest(step.Method, url, bytes.NewReader([]byte(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Process and set headers
	for key, value := range step.Headers {
		processedValue := template.ProcessTemplateWithContext(value, exch, nil)
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
	exch.Response = &exchange.ResponseContext{
		Response: resp,
		Body:     respBody,
	}

	// Process captures using the common capture logic
	if len(step.Capture) > 0 {
		capture.CaptureRequestData(imposterConfig, step.Capture, exch)
	}

	return nil
}
