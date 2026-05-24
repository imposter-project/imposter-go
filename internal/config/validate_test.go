package config

import "testing"

func TestValidateUpstreams(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid upstream and passthrough reference",
			cfg: Config{
				Upstreams: map[string]Upstream{"backend": {URL: "http://api.example.com"}},
				Resources: []Resource{{BaseResource: BaseResource{Passthrough: "backend"}}},
			},
		},
		{
			name: "unknown upstream reference",
			cfg: Config{
				Upstreams: map[string]Upstream{"backend": {URL: "http://api.example.com"}},
				Resources: []Resource{{BaseResource: BaseResource{Passthrough: "missing"}}},
			},
			wantErr: true,
		},
		{
			name: "upstream URL missing scheme",
			cfg: Config{
				Upstreams: map[string]Upstream{"backend": {URL: "api.example.com"}},
			},
			wantErr: true,
		},
		{
			name: "no passthrough config is valid",
			cfg:  Config{Resources: []Resource{{BaseResource: BaseResource{}}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.cfg)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
