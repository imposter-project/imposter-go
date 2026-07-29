package rest

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
)

// runScriptPrecedenceRequest exercises the full pipeline for a single resource
// combining an inline script step with a response configuration, and returns
// the resulting response state.
func runScriptPrecedenceRequest(
	t *testing.T,
	scriptCode string,
	resp *config.Response,
	files map[string]string,
) *exchange.ResponseState {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "imposter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	cfg := &config.Config{
		Plugin:    "rest",
		ConfigDir: tempDir,
		Resources: []config.Resource{
			{
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{Method: "POST", Path: "/orders"},
					Steps: []config.Step{
						{Type: config.ScriptStepType, Lang: "javascript", Code: scriptCode},
					},
					Response: resp,
				},
			},
		},
	}

	handler, err := NewPluginHandler(cfg, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("POST", "/orders", bytes.NewReader([]byte(`{}`)))
	responseState := response.NewResponseState()
	exch := exchange.NewExchange(req, []byte(`{}`), store.NewRequestStore(), responseState)
	handler.HandleRequest(exch, response.NewProcessor(&config.ImposterConfig{}, tempDir))

	return responseState
}

// TestHandler_ScriptStatusCodeBeatsConfig checks that a status code set by a
// script is not overwritten by the resource's response configuration, matching
// the behaviour of the JVM engine.
func TestHandler_ScriptStatusCodeBeatsConfig(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withStatusCode(409)`,
		&config.Response{StatusCode: 200, Content: `{"ok":true}`},
		nil,
	)

	if rs.StatusCode != http.StatusConflict {
		t.Errorf("Expected script status code %d, got %d", http.StatusConflict, rs.StatusCode)
	}
	// the script did not set any content, so the configured content applies
	if string(rs.Body) != `{"ok":true}` {
		t.Errorf("Expected configured content, got %q", string(rs.Body))
	}
}

// TestHandler_ScriptContentBeatsConfig checks that content set by a script is
// not overwritten by the resource's configured content.
func TestHandler_ScriptContentBeatsConfig(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withContent('{"error":"conflict"}')`,
		&config.Response{StatusCode: 400, Content: `{"ok":true}`},
		nil,
	)

	if string(rs.Body) != `{"error":"conflict"}` {
		t.Errorf("Expected script content, got %q", string(rs.Body))
	}
	// the script did not set a status code, so the configured one applies
	if rs.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected configured status code %d, got %d", http.StatusBadRequest, rs.StatusCode)
	}
}

// TestHandler_ScriptContentBeatsConfigFile checks that script-set content takes
// precedence over a configured response file, which acts only as a default.
func TestHandler_ScriptContentBeatsConfigFile(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withContent('from-script')`,
		&config.Response{File: "body.json"},
		map[string]string{"body.json": "from-file"},
	)

	if string(rs.Body) != "from-script" {
		t.Errorf("Expected script content, got %q", string(rs.Body))
	}
}

// TestHandler_ScriptFileBeatsScriptContent checks that a response file set by
// the script wins over content set by the same script, as the file is the more
// specific source.
func TestHandler_ScriptFileBeatsScriptContent(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withContent('from-script-content').withFile('script.json')`,
		&config.Response{},
		map[string]string{"script.json": "from-script-file"},
	)

	if string(rs.Body) != "from-script-file" {
		t.Errorf("Expected script file content, got %q", string(rs.Body))
	}
}

// TestHandler_ScriptContentBeatsConfigDir checks that script-set content takes
// precedence over a directory-based response.
func TestHandler_ScriptContentBeatsConfigDir(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withContent('from-script')`,
		&config.Response{Dir: "."},
		nil,
	)

	if string(rs.Body) != "from-script" {
		t.Errorf("Expected script content, got %q", string(rs.Body))
	}
}

// TestHandler_ScriptFileBeatsConfigFile checks that a response file set by a
// script takes precedence over the configured file.
func TestHandler_ScriptFileBeatsConfigFile(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withFile('script.json')`,
		&config.Response{File: "config.json"},
		map[string]string{"script.json": "from-script-file", "config.json": "from-config-file"},
	)

	if string(rs.Body) != "from-script-file" {
		t.Errorf("Expected script file content, got %q", string(rs.Body))
	}
}

// TestHandler_ScriptEmptyBodyNotRefilledFromConfig checks that withEmpty() is
// honoured rather than being replaced by the configured content.
func TestHandler_ScriptEmptyBodyNotRefilledFromConfig(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withEmpty()`,
		&config.Response{Content: `{"ok":true}`},
		nil,
	)

	if len(rs.Body) != 0 {
		t.Errorf("Expected empty body, got %q", string(rs.Body))
	}
}

// TestHandler_ScriptHeadersBeatConfigHeaders checks that script-set headers
// take precedence over configured headers of the same name, and that headers
// set by only one of the two are retained.
func TestHandler_ScriptHeadersBeatConfigHeaders(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withHeader("X-Shared", "script").withHeader("X-Script-Only", "script")`,
		&config.Response{Headers: map[string]string{"X-Shared": "config", "X-Config-Only": "config"}},
		nil,
	)

	if rs.Headers["X-Shared"] != "script" {
		t.Errorf("Expected script header to win, got %q", rs.Headers["X-Shared"])
	}
	if rs.Headers["X-Script-Only"] != "script" {
		t.Errorf("Expected script-only header to be retained, got %q", rs.Headers["X-Script-Only"])
	}
	if rs.Headers["X-Config-Only"] != "config" {
		t.Errorf("Expected configured-only header to be applied, got %q", rs.Headers["X-Config-Only"])
	}
}

// TestHandler_ScriptHeadersBeatConfigHeadersIgnoringCase checks that a
// configured header does not override a script-set header that differs only in
// casing, since HTTP header names are case-insensitive.
func TestHandler_ScriptHeadersBeatConfigHeadersIgnoringCase(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withHeader("X-Shared", "script")`,
		&config.Response{Headers: map[string]string{"x-shared": "config"}},
		nil,
	)

	if rs.Headers["X-Shared"] != "script" {
		t.Errorf("Expected script header to win, got %q", rs.Headers["X-Shared"])
	}
	if _, exists := rs.Headers["x-shared"]; exists {
		t.Errorf("Expected configured header to be skipped, got %q", rs.Headers["x-shared"])
	}
}

// TestHandler_ScriptContentIsTemplated checks that script-set content passes
// through the template processor when the response configuration enables
// templating.
func TestHandler_ScriptContentIsTemplated(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withContent('path: ${context.request.path}')`,
		&config.Response{Template: true},
		nil,
	)

	if string(rs.Body) != "path: /orders" {
		t.Errorf("Expected script content to be templated, got %q", string(rs.Body))
	}
}

// TestHandler_ScriptStatusCodeSurvivesEmptyResponseBlock checks that an empty
// response block does not reset a script-set status code.
func TestHandler_ScriptStatusCodeSurvivesEmptyResponseBlock(t *testing.T) {
	rs := runScriptPrecedenceRequest(t,
		`respond().withStatusCode(201).withContent('created')`,
		&config.Response{},
		nil,
	)

	if rs.StatusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rs.StatusCode)
	}
	if string(rs.Body) != "created" {
		t.Errorf("Expected script content, got %q", string(rs.Body))
	}
}

// TestHandler_InterceptorScriptValuesSurviveResourceConfig checks that values
// set by an interceptor's script are not overwritten by the matched resource's
// response configuration, mirroring the JVM engine, where the response
// behaviour persists across handlers for the exchange.
func TestHandler_InterceptorScriptValuesSurviveResourceConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "imposter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		Plugin:    "rest",
		ConfigDir: tempDir,
		Interceptors: []config.Interceptor{
			{
				Continue: true,
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{Method: "POST", Path: "/orders"},
					Steps: []config.Step{
						{
							Type: config.ScriptStepType,
							Lang: "javascript",
							Code: `respond().withStatusCode(429)`,
						},
					},
				},
			},
		},
		Resources: []config.Resource{
			{
				BaseResource: config.BaseResource{
					RequestMatcher: config.RequestMatcher{Method: "POST", Path: "/orders"},
					Response:       &config.Response{StatusCode: 200, Content: `{"ok":true}`},
				},
			},
		},
	}

	handler, err := NewPluginHandler(cfg, &config.ImposterConfig{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("POST", "/orders", bytes.NewReader([]byte(`{}`)))
	rs := response.NewResponseState()
	exch := exchange.NewExchange(req, []byte(`{}`), store.NewRequestStore(), rs)
	handler.HandleRequest(exch, response.NewProcessor(&config.ImposterConfig{}, tempDir))

	if rs.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected interceptor script status code %d, got %d", http.StatusTooManyRequests, rs.StatusCode)
	}
	if string(rs.Body) != `{"ok":true}` {
		t.Errorf("Expected configured content, got %q", string(rs.Body))
	}
}
