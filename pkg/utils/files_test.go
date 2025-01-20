package utils

import (
	"path/filepath"
	"testing"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		configDir string
		want      string
		wantErr   bool
	}{
		{
			name:      "valid path within config directory",
			path:      "test.json",
			configDir: "/config",
			want:      filepath.Join("/config", "test.json"),
			wantErr:   false,
		},
		{
			name:      "valid nested path within config directory",
			path:      "nested/test.json",
			configDir: "/config",
			want:      filepath.Join("/config", "nested/test.json"),
			wantErr:   false,
		},
		{
			name:      "path attempting directory traversal",
			path:      "../test.json",
			configDir: "/config",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "path attempting deep directory traversal",
			path:      "../../etc/passwd",
			configDir: "/config",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "absolute path within config directory",
			path:      "/test.json",
			configDir: "/config",
			want:      filepath.Join("/config", "test.json"),
			wantErr:   false,
		},
		{
			name:      "path with dot-dot that resolves within config directory",
			path:      "nested/../test.json",
			configDir: "/config",
			want:      filepath.Join("/config", "test.json"),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePath(tt.path, tt.configDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidatePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
