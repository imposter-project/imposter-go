package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/handler"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/plugin"
	"github.com/stretchr/testify/require"
)

// startWebSocketServer loads the given config content and starts a test
// server, returning the ws:// URL for the given path.
func startWebSocketServer(t *testing.T, configContent string) *httptest.Server {
	t.Helper()
	tempDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tempDir, "ws-config.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	store.InitStoreProvider()
	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins, err := plugin.LoadPlugins(configs, imposterConfig, nil)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(imposterConfig, w, r, plugins)
	}))
	t.Cleanup(server.Close)
	return server
}

func wsURL(server *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(server.URL, "http") + path
}

// readTextMessage reads a single text frame with a timeout.
func readTextMessage(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	messageType, data, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, websocket.TextMessage, messageType)
	return string(data)
}

func TestWebSocket_OpenGreetingAndMessageMatching(t *testing.T) {
	configContent := `plugin: websocket
resources:
  - path: /gateway
    on: open
    response:
      content: '{"type":"event","event":"connect.challenge","payload":{"nonce":"${random.uuid()}"}}'
      template: true

  - path: /gateway
    requestBody:
      allOf:
        - jsonPath: $.type
          value: req
        - jsonPath: $.method
          value: connect
    capture:
      reqId:
        requestBody:
          jsonPath: $.id
    response:
      content: '{"type":"res","id":"${stores.request.reqId}","ok":true}'
      template: true
`
	server := startWebSocketServer(t, configContent)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server, "/gateway"), nil)
	require.NoError(t, err)
	defer conn.Close()

	// The open greeting arrives first, with the template processed
	greeting := readTextMessage(t, conn)
	var greetingPayload struct {
		Type    string `json:"type"`
		Event   string `json:"event"`
		Payload struct {
			Nonce string `json:"nonce"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal([]byte(greeting), &greetingPayload))
	require.Equal(t, "event", greetingPayload.Type)
	require.Equal(t, "connect.challenge", greetingPayload.Event)
	require.Len(t, greetingPayload.Payload.Nonce, 36) // templated UUID

	// A matched message gets a reply echoing the captured request ID
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"req","id":"abc-123","method":"connect"}`))
	require.NoError(t, err)

	reply := readTextMessage(t, conn)
	require.JSONEq(t, `{"type":"res","id":"abc-123","ok":true}`, reply)
}

func TestWebSocket_StreamedResponses(t *testing.T) {
	configContent := `plugin: websocket
resources:
  - path: /gateway
    requestBody:
      jsonPath: $.method
      value: agent
    capture:
      reqId:
        requestBody:
          jsonPath: $.id
    responses:
      - content: '{"type":"res","id":"${stores.request.reqId}","ok":true}'
        template: true
      - content: '{"type":"event","event":"agent","payload":{"seq":1}}'
        delay:
          exact: 200
      - content: '{"type":"event","event":"agent","payload":{"seq":2}}'
`
	server := startWebSocketServer(t, configContent)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server, "/gateway"), nil)
	require.NoError(t, err)
	defer conn.Close()

	start := time.Now()
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"req","id":"req-9","method":"agent"}`))
	require.NoError(t, err)

	first := readTextMessage(t, conn)
	require.JSONEq(t, `{"type":"res","id":"req-9","ok":true}`, first)

	second := readTextMessage(t, conn)
	require.Contains(t, second, `"seq":1`)
	// The second frame is delayed by 200ms from the first
	require.GreaterOrEqual(t, time.Since(start), 200*time.Millisecond)

	third := readTextMessage(t, conn)
	require.Contains(t, third, `"seq":2`)
}

func TestWebSocket_ConnectionScopedSchedule(t *testing.T) {
	configContent := `plugin: websocket
resources:
  - path: /gateway
    on: open
    schedule:
      - every: 100ms
        response:
          content: '{"type":"event","event":"tick"}'
`
	server := startWebSocketServer(t, configContent)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server, "/gateway"), nil)
	require.NoError(t, err)
	defer conn.Close()

	// At least two ticks arrive
	for i := 0; i < 2; i++ {
		tick := readTextMessage(t, conn)
		require.Contains(t, tick, `"tick"`)
	}
}

func TestWebSocket_UnmatchedPathAndPlainRequests(t *testing.T) {
	configContent := `plugin: websocket
resources:
  - path: /gateway
    on: open
    response:
      content: hello
`
	server := startWebSocketServer(t, configContent)

	// Upgrade to a non-matching path is refused
	_, resp, err := websocket.DefaultDialer.Dial(wsURL(server, "/other"), nil)
	require.Error(t, err)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// A plain HTTP GET to the websocket path is not handled by the plugin
	httpResp, err := http.Get(server.URL + "/gateway")
	require.NoError(t, err)
	defer httpResp.Body.Close()
	require.Equal(t, http.StatusNotFound, httpResp.StatusCode)
}

func TestWebSocket_CoexistsWithRestPlugin(t *testing.T) {
	tempDir := t.TempDir()

	wsConfig := `plugin: websocket
resources:
  - path: /ws
    on: open
    response:
      content: ws-hello
`
	restConfig := `plugin: rest
resources:
  - path: /http
    method: GET
    response:
      content: rest-hello
`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "ws-config.yaml"), []byte(wsConfig), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "rest-config.yaml"), []byte(restConfig), 0644))

	imposterConfig := config.LoadImposterConfig()
	configs := config.LoadConfig(tempDir, imposterConfig)
	plugins, err := plugin.LoadPlugins(configs, imposterConfig, nil)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(imposterConfig, w, r, plugins)
	}))
	defer server.Close()

	// REST route works
	httpResp, err := http.Get(server.URL + "/http")
	require.NoError(t, err)
	defer httpResp.Body.Close()
	require.Equal(t, http.StatusOK, httpResp.StatusCode)

	// WebSocket route works on the same port
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server, "/ws"), nil)
	require.NoError(t, err)
	defer conn.Close()
	require.Equal(t, "ws-hello", readTextMessage(t, conn))
}

func TestWebSocket_CloseResourceRunsSteps(t *testing.T) {
	configContent := `plugin: websocket
resources:
  - path: /gateway
    on: open
    response:
      content: hello

  - path: /gateway
    on: close
    steps:
      - type: script
        lang: javascript
        code: |
          var s = stores.open('closes');
          s.save('lastClose', 'yes');
`
	server := startWebSocketServer(t, configContent)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server, "/gateway"), nil)
	require.NoError(t, err)
	require.Equal(t, "hello", readTextMessage(t, conn))
	require.NoError(t, conn.Close())

	// The close resource's steps run asynchronously after disconnect
	require.Eventually(t, func() bool {
		resp, err := http.Get(server.URL + "/system/store/closes/lastClose")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 50*time.Millisecond)
}

func TestWebSocket_ScheduleLimit(t *testing.T) {
	configContent := `plugin: websocket
resources:
  - path: /gateway
    on: open
    schedule:
      - every: 50ms
        limit: 2
        response:
          content: '{"type":"event","event":"tick"}'
`
	server := startWebSocketServer(t, configContent)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server, "/gateway"), nil)
	require.NoError(t, err)
	defer conn.Close()

	// Exactly two ticks arrive, then the schedule stops
	for i := 0; i < 2; i++ {
		tick := readTextMessage(t, conn)
		require.Contains(t, tick, `"tick"`)
	}

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(500*time.Millisecond)))
	_, _, err = conn.ReadMessage()
	require.Error(t, err, "expected no further ticks after the limit was reached")
}

func TestWebSocket_WildcardPathAcceptsAnyPath(t *testing.T) {
	// A resource distinguished only by 'on' and a wildcard path must still
	// match: the connection upgrades AND the open event is handled. Mirrors
	// the real OpenClaw gateway, which accepts the upgrade on any path.
	configContent := `plugin: websocket
resources:
  - path: /*
    on: open
    response:
      content: '{"event":"welcome"}'
`
	server := startWebSocketServer(t, configContent)

	for _, path := range []string{"/ws", "/gateway", "/anything/else"} {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL(server, path), nil)
		require.NoErrorf(t, err, "dialing %s", path)
		require.JSONEq(t, `{"event":"welcome"}`, readTextMessage(t, conn))
		conn.Close()
	}
}
