package websocket

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/pipeline"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

const (
	// outBufferSize bounds the outbound frame queue; producers block (until
	// the connection closes) once a slow client falls this far behind.
	outBufferSize = 64

	closeWriteTimeout = time.Second
)

// wsConn represents a live mock websocket connection.
type wsConn struct {
	handler *PluginHandler
	conn    *websocket.Conn
	upgrade *http.Request

	// requestStore is connection-scoped: values captured from one message are
	// visible to resources matching later messages on the same connection.
	requestStore *store.Store

	out    chan []byte
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newConnection(h *PluginHandler, conn *websocket.Conn, upgrade *http.Request) *wsConn {
	ctx, cancel := context.WithCancel(context.Background())
	return &wsConn{
		handler:      h,
		conn:         conn,
		upgrade:      upgrade,
		requestStore: store.NewRequestStore(),
		out:          make(chan []byte, outBufferSize),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// run drives the connection: it emits any 'open' responses, starts
// connection-scoped schedules, then reads messages until the client
// disconnects, finally running any 'close' resources.
func (c *wsConn) run() {
	c.wg.Add(1)
	go c.writeLoop()

	// Run the open pipeline synchronously so its responses are sent before
	// any message replies or scheduled frames.
	openState := c.runEventPipeline(config.WebSocketEventOpen, nil)
	if openState.Resource != nil {
		c.startSchedules(openState.Resource)
	}

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			logger.Debugf("websocket connection closed - path:%s, reason:%v", c.upgrade.URL.Path, err)
			break
		}
		if messageType != websocket.TextMessage {
			logger.Debugf("ignoring non-text websocket message - path:%s, type:%d", c.upgrade.URL.Path, messageType)
			continue
		}
		if logger.IsTraceEnabled() {
			logger.Tracef("websocket message received - path:%s, body:%s", c.upgrade.URL.Path, data)
		}
		c.runEventPipeline(config.WebSocketEventMessage, data)
	}

	// The connection is gone; no frames can be sent from close resources,
	// but their captures and steps still run.
	c.runEventPipeline(config.WebSocketEventClose, nil)

	c.cancel()
	c.wg.Wait()
	_ = c.conn.Close()
	logger.Infof("websocket connection closed - path:%s", c.upgrade.URL.Path)
}

// writeLoop is the single writer for the connection; gorilla/websocket
// forbids concurrent writers, and frames may be produced by both message
// replies and schedule firings.
func (c *wsConn) writeLoop() {
	defer c.wg.Done()
	for {
		select {
		case msg := <-c.out:
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				logger.Debugf("websocket write failed - path:%s, error:%v", c.upgrade.URL.Path, err)
				c.cancel()
				return
			}
		case <-c.ctx.Done():
			deadline := time.Now().Add(closeWriteTimeout)
			_ = c.conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), deadline)
			return
		}
	}
}

// send enqueues a text frame for the writer goroutine, giving up if the
// connection closes first.
func (c *wsConn) send(data []byte) {
	msg := make([]byte, len(data))
	copy(msg, data)
	select {
	case c.out <- msg:
	case <-c.ctx.Done():
	}
}

// runEventPipeline runs the shared request pipeline for a connection event,
// using the original upgrade request for handshake matching and the message
// payload (if any) as the request body. Each response block produced by the
// matched resource is sent as a text frame, except for 'close' events.
func (c *wsConn) runEventPipeline(event string, body []byte) *exchange.ResponseState {
	h := c.handler
	responseState := response.NewResponseState()
	exch := exchange.NewExchange(c.upgrade, body, c.requestStore, responseState)

	hooks := &pipeline.ProtocolHooks{
		CalculateScore: func(exch *exchange.Exchange, reqMatcher *config.RequestMatcher,
			systemNamespaces map[string]string, imposterConfig *config.ImposterConfig,
		) (int, bool) {
			if reqMatcher.NormalisedOn() != event {
				return matcher.NegativeMatchScore, false
			}
			return matcher.CalculateMatchScore(exch, reqMatcher, systemNamespaces, imposterConfig)
		},
		OnStepError: func(rs *exchange.ResponseState, msg string) {
			logger.Errorf("websocket %s handler steps failed - path:%s: %s", event, c.upgrade.URL.Path, msg)
			rs.Handled = true
		},
		ProcessResponse: func(exch *exchange.Exchange, reqMatcher *config.RequestMatcher,
			resp *config.Response, respProc response.Processor,
		) {
			c.processAndSend(exch, reqMatcher, resp, event != config.WebSocketEventClose)
		},
		GetResourceName: func(resource *config.Resource) (string, string) {
			return resource.Path, "WS"
		},
	}

	pipeline.RunPipeline(h.config, h.imposterConfig, exch, h.respProc, hooks)

	if !responseState.Handled && event == config.WebSocketEventMessage {
		logger.Debugf("no resource matched websocket message - path:%s", c.upgrade.URL.Path)
	}
	return responseState
}

// processAndSend runs standard response processing (delay, file/content
// resolution, templating) and enqueues the result as a text frame.
func (c *wsConn) processAndSend(exch *exchange.Exchange, reqMatcher *config.RequestMatcher,
	resp *config.Response, sendFrame bool,
) {
	c.handler.respProc(exch, reqMatcher, resp)

	rs := exch.ResponseState
	if sendFrame && len(rs.Body) > 0 {
		c.send(rs.Body)
	}
	// Reset per-response fields so subsequent responses in a 'responses'
	// list start clean.
	rs.Body = nil
	rs.File = ""
}
