package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrefixResponseFiles(t *testing.T) {
	t.Run("prefixes file and dir", func(t *testing.T) {
		resp := &Response{File: "data.json", Dir: "responses"}
		prefixResponseFiles(resp, "subdir")
		require.Equal(t, filepath.Join("subdir", "data.json"), resp.File)
		require.Equal(t, filepath.Join("subdir", "responses"), resp.Dir)
	})

	t.Run("no-op when relDir is current directory", func(t *testing.T) {
		resp := &Response{File: "data.json", Dir: "responses"}
		prefixResponseFiles(resp, ".")
		require.Equal(t, "data.json", resp.File)
		require.Equal(t, "responses", resp.Dir)
	})

	t.Run("no-op on nil response", func(t *testing.T) {
		require.NotPanics(t, func() {
			prefixResponseFiles(nil, "subdir")
		})
	})

	t.Run("leaves empty fields empty", func(t *testing.T) {
		resp := &Response{Content: "inline"}
		prefixResponseFiles(resp, "subdir")
		require.Empty(t, resp.File)
		require.Empty(t, resp.Dir)
	})
}

func TestPrefixStepFiles(t *testing.T) {
	t.Run("prefixes step files", func(t *testing.T) {
		steps := []Step{
			{Type: ScriptStepType, File: "script.js"},
			{Type: ScriptStepType, Code: "console.log('inline')"},
			{Type: RemoteStepType, URL: "http://example.com"},
		}
		prefixStepFiles(steps, "subdir")
		require.Equal(t, filepath.Join("subdir", "script.js"), steps[0].File)
		require.Empty(t, steps[1].File)
		require.Empty(t, steps[2].File)
	})

	t.Run("no-op when relDir is current directory", func(t *testing.T) {
		steps := []Step{{Type: ScriptStepType, File: "script.js"}}
		prefixStepFiles(steps, ".")
		require.Equal(t, "script.js", steps[0].File)
	})
}

func TestPrefixScheduleFiles(t *testing.T) {
	t.Run("prefixes response, responses and step files", func(t *testing.T) {
		schedules := []Schedule{
			{
				Every:    "30s",
				Response: &Response{File: "tick.json"},
				Steps:    []Step{{Type: ScriptStepType, File: "job.js"}},
			},
			{
				Cron: "0 * * * *",
				Responses: []Response{
					{File: "first.json"},
					{Content: "inline"},
					{File: "second.json"},
				},
			},
		}
		prefixScheduleFiles(schedules, "subdir")

		require.Equal(t, filepath.Join("subdir", "tick.json"), schedules[0].Response.File)
		require.Equal(t, filepath.Join("subdir", "job.js"), schedules[0].Steps[0].File)
		require.Equal(t, filepath.Join("subdir", "first.json"), schedules[1].Responses[0].File)
		require.Empty(t, schedules[1].Responses[1].File)
		require.Equal(t, filepath.Join("subdir", "second.json"), schedules[1].Responses[2].File)
	})

	t.Run("no-op when relDir is current directory", func(t *testing.T) {
		schedules := []Schedule{{
			Every:    "30s",
			Response: &Response{File: "tick.json"},
			Steps:    []Step{{Type: ScriptStepType, File: "job.js"}},
		}}
		prefixScheduleFiles(schedules, ".")
		require.Equal(t, "tick.json", schedules[0].Response.File)
		require.Equal(t, "job.js", schedules[0].Steps[0].File)
	})
}
