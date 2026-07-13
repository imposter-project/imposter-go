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

func TestValidateWebSocketAndSchedules(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "websocket resource with on and responses",
			cfg: Config{
				Plugin: "websocket",
				Resources: []Resource{{BaseResource: BaseResource{
					RequestMatcher: RequestMatcher{Path: "/ws", On: "open"},
					Responses:      []Response{{Content: "a"}, {Content: "b"}},
				}}},
			},
		},
		{
			name: "both response and responses is an error",
			cfg: Config{
				Plugin: "websocket",
				Resources: []Resource{{BaseResource: BaseResource{
					RequestMatcher: RequestMatcher{Path: "/ws"},
					Response:       &Response{Content: "a"},
					Responses:      []Response{{Content: "b"}},
				}}},
			},
			wantErr: true,
		},
		{
			name: "on outside websocket plugin is an error",
			cfg: Config{
				Plugin: "rest",
				Resources: []Resource{{BaseResource: BaseResource{
					RequestMatcher: RequestMatcher{Path: "/a", On: "open"},
				}}},
			},
			wantErr: true,
		},
		{
			name: "responses outside websocket plugin is an error",
			cfg: Config{
				Plugin: "rest",
				Resources: []Resource{{BaseResource: BaseResource{
					RequestMatcher: RequestMatcher{Path: "/a"},
					Responses:      []Response{{Content: "b"}},
				}}},
			},
			wantErr: true,
		},
		{
			name: "invalid on value",
			cfg: Config{
				Plugin: "websocket",
				Resources: []Resource{{BaseResource: BaseResource{
					RequestMatcher: RequestMatcher{Path: "/ws", On: "bogus"},
				}}},
			},
			wantErr: true,
		},
		{
			name: "schedule on non-open resource is an error",
			cfg: Config{
				Plugin: "websocket",
				Resources: []Resource{{BaseResource: BaseResource{
					RequestMatcher: RequestMatcher{Path: "/ws"},
					Schedule:       []Schedule{{Every: "15s", Response: &Response{Content: "tick"}}},
				}}},
			},
			wantErr: true,
		},
		{
			name: "connection schedule on open resource",
			cfg: Config{
				Plugin: "websocket",
				Resources: []Resource{{BaseResource: BaseResource{
					RequestMatcher: RequestMatcher{Path: "/ws", On: "open"},
					Schedule:       []Schedule{{Every: "15s", Response: &Response{Content: "tick"}}},
				}}},
			},
		},
		{
			name: "top-level schedule with steps and every",
			cfg: Config{
				Plugin:    "rest",
				Schedules: []Schedule{{Name: "job", Every: "30s", Steps: []Step{{Type: RemoteStepType, URL: "http://example.com"}}}},
			},
		},
		{
			name: "top-level schedule with cron",
			cfg: Config{
				Plugin:    "rest",
				Schedules: []Schedule{{Cron: "0 * * * *", Steps: []Step{{Type: RemoteStepType, URL: "http://example.com"}}}},
			},
		},
		{
			name: "schedule with both every and cron is an error",
			cfg: Config{
				Plugin:    "rest",
				Schedules: []Schedule{{Every: "30s", Cron: "0 * * * *", Steps: []Step{{Type: RemoteStepType}}}},
			},
			wantErr: true,
		},
		{
			name: "schedule with neither every nor cron is an error",
			cfg: Config{
				Plugin:    "rest",
				Schedules: []Schedule{{Steps: []Step{{Type: RemoteStepType}}}},
			},
			wantErr: true,
		},
		{
			name: "schedule with invalid every duration is an error",
			cfg: Config{
				Plugin:    "rest",
				Schedules: []Schedule{{Every: "nonsense", Steps: []Step{{Type: RemoteStepType}}}},
			},
			wantErr: true,
		},
		{
			name: "schedule with invalid cron expression is an error",
			cfg: Config{
				Plugin:    "rest",
				Schedules: []Schedule{{Cron: "not a cron", Steps: []Step{{Type: RemoteStepType}}}},
			},
			wantErr: true,
		},
		{
			name: "top-level schedule without steps is an error",
			cfg: Config{
				Plugin:    "rest",
				Schedules: []Schedule{{Every: "30s"}},
			},
			wantErr: true,
		},
		{
			name: "top-level schedule with response is an error",
			cfg: Config{
				Plugin:    "rest",
				Schedules: []Schedule{{Every: "30s", Steps: []Step{{Type: RemoteStepType}}, Response: &Response{Content: "x"}}},
			},
			wantErr: true,
		},
		{
			name: "websocket passthrough is an error",
			cfg: Config{
				Plugin:    "websocket",
				Upstreams: map[string]Upstream{"backend": {URL: "http://api.example.com"}},
				Resources: []Resource{{BaseResource: BaseResource{
					RequestMatcher: RequestMatcher{Path: "/ws"},
					Passthrough:    "backend",
				}}},
			},
			wantErr: true,
		},
		{
			name: "interceptor with responses is an error",
			cfg: Config{
				Plugin: "rest",
				Interceptors: []Interceptor{{BaseResource: BaseResource{
					Responses: []Response{{Content: "a"}},
				}}},
			},
			wantErr: true,
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

func TestEffectiveResponses(t *testing.T) {
	t.Run("singular response equals one-element responses list", func(t *testing.T) {
		res := BaseResource{Response: &Response{Content: "a"}}
		resps := res.EffectiveResponses()
		if len(resps) != 1 || resps[0].Content != "a" {
			t.Fatalf("unexpected responses: %+v", resps)
		}
	})

	t.Run("responses list is returned as-is", func(t *testing.T) {
		res := BaseResource{Responses: []Response{{Content: "a"}, {Content: "b"}}}
		resps := res.EffectiveResponses()
		if len(resps) != 2 {
			t.Fatalf("unexpected responses: %+v", resps)
		}
	})

	t.Run("no responses yields nil", func(t *testing.T) {
		res := BaseResource{}
		if resps := res.EffectiveResponses(); resps != nil {
			t.Fatalf("expected nil, got: %+v", resps)
		}
	})
}

func TestValidateScheduleLimit(t *testing.T) {
	valid := Config{
		Plugin:    "rest",
		Schedules: []Schedule{{Every: "30s", Limit: 10, Steps: []Step{{Type: RemoteStepType}}}},
	}
	if err := Validate(&valid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	negative := Config{
		Plugin:    "rest",
		Schedules: []Schedule{{Every: "30s", Limit: -1, Steps: []Step{{Type: RemoteStepType}}}},
	}
	if err := Validate(&negative); err == nil {
		t.Fatal("expected error for negative limit, got nil")
	}
}
