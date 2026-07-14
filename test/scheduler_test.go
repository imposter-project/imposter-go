package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/scheduler"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/stretchr/testify/require"
)

func TestScheduler_PeriodicWebhook(t *testing.T) {
	// A counter server standing in for the webhook receiver
	var hits atomic.Int32
	var lastBody atomic.Value
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		lastBody.Store(string(body))
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	tempDir := t.TempDir()
	configContent := `plugin: rest
resources:
  - path: /
    method: GET
    response:
      content: ok
schedules:
  - name: test-webhook
    every: 100ms
    steps:
      - type: remote
        url: ` + receiver.URL + `/webhook
        method: POST
        body: '{"event":"test"}'
`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "sched-config.yaml"), []byte(configContent), 0644))

	store.InitStoreProvider()
	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	require.Len(t, configs, 1)

	scheduler.Start([]*config.Config{&configs[0]}, imposterConfig)

	// At least two firings occur
	require.Eventually(t, func() bool {
		return hits.Load() >= 2
	}, 5*time.Second, 20*time.Millisecond)

	require.JSONEq(t, `{"event":"test"}`, lastBody.Load().(string))

	// Stop halts further firings
	scheduler.Stop()
	after := hits.Load()
	time.Sleep(300 * time.Millisecond)
	require.Equal(t, after, hits.Load())
}
