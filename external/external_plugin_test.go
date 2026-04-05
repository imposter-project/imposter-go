package external

import (
	"strings"
	"testing"

	"github.com/imposter-project/imposter-go/external/shared"
	"github.com/imposter-project/imposter-go/internal/exchange"
)

// TestApplyTransformResult_FileNameInfersContentType is a regression test for a
// bug where static assets served by external plugins (e.g. Swagger UI CSS/JS)
// were sent with the default text/plain content type, causing browsers to
// refuse to apply the stylesheets. The plugin sets a FileName hint on the
// TransformResponseResult and applyTransformResult must infer Content-Type
// from its extension when the plugin has not set one explicitly.
func TestApplyTransformResult_FileNameInfersContentType(t *testing.T) {
	tests := []struct {
		name                string
		result              shared.TransformResponseResult
		existingContentType string
		wantContentType     string
	}{
		{
			name: "css file name infers text/css",
			result: shared.TransformResponseResult{
				StatusCode: 200,
				Body:       []byte("body { color: red; }"),
				FileName:   "swagger-ui.css",
			},
			wantContentType: "text/css",
		},
		{
			name: "js file name infers javascript content type",
			result: shared.TransformResponseResult{
				StatusCode: 200,
				Body:       []byte("console.log('hi');"),
				FileName:   "swagger-ui-bundle.js",
			},
			wantContentType: "javascript",
		},
		{
			name: "html file name infers text/html",
			result: shared.TransformResponseResult{
				StatusCode: 200,
				Body:       []byte("<html></html>"),
				FileName:   "index.html",
			},
			wantContentType: "text/html",
		},
		{
			name: "yaml file name infers application/x-yaml",
			result: shared.TransformResponseResult{
				StatusCode: 200,
				Body:       []byte("key: value"),
				FileName:   "petstore.yaml",
			},
			wantContentType: "application/x-yaml",
		},
		{
			name: "empty file name leaves content type unset",
			result: shared.TransformResponseResult{
				StatusCode: 200,
				Body:       []byte("plain body"),
			},
			wantContentType: "",
		},
		{
			name: "explicit content type is not overridden",
			result: shared.TransformResponseResult{
				StatusCode: 200,
				Body:       []byte("{}"),
				Headers:    map[string]string{"Content-Type": "application/json"},
				FileName:   "data.yaml",
			},
			wantContentType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &exchange.ResponseState{Headers: map[string]string{}}
			applyTransformResult(tt.result, rs)

			got := rs.Headers["Content-Type"]
			if tt.wantContentType == "" {
				if got != "" {
					t.Errorf("expected no Content-Type, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantContentType) {
				t.Errorf("expected Content-Type to contain %q, got %q", tt.wantContentType, got)
			}
		})
	}
}
