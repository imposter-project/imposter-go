package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/matcher"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// HandleRequest upgrades matching websocket handshake requests and hands the
// connection to a per-connection goroutine. Non-upgrade requests, and upgrade
// requests whose path matches no resource, are left unhandled for other
// plugins (or the 404 handler).
func (h *PluginHandler) HandleRequest(exch *exchange.Exchange, respProc response.Processor) {
	r := exch.Request.Request
	responseState := exch.ResponseState

	if !websocket.IsWebSocketUpgrade(r) {
		return
	}

	if !h.pathMatches(exch) {
		return
	}

	if exch.ResponseWriter == nil {
		logger.Warnf("websocket upgrade requested but no response writer is available - path:%s", r.URL.Path)
		responseState.StatusCode = http.StatusNotImplemented
		responseState.Body = []byte("websocket upgrade is not supported in this runtime")
		responseState.Handled = true
		return
	}
	if _, ok := exch.ResponseWriter.(http.Hijacker); !ok {
		logger.Warnf("websocket upgrade requires HTTP/1.1 connection hijacking, which the underlying connection does not support - path:%s", r.URL.Path)
		responseState.StatusCode = http.StatusNotImplemented
		responseState.Body = []byte("websocket upgrade requires an HTTP/1.1 connection")
		responseState.Handled = true
		return
	}

	conn, err := h.upgrader.Upgrade(exch.ResponseWriter, r, nil)
	if err != nil {
		logger.Warnf("websocket upgrade failed - path:%s, error:%v", r.URL.Path, err)
		responseState.StatusCode = http.StatusBadRequest
		responseState.Body = []byte("websocket upgrade failed")
		responseState.Handled = true
		return
	}

	logger.Infof("websocket connection opened - path:%s", r.URL.Path)
	responseState.Hijacked = true
	responseState.Handled = true

	wsc := newConnection(h, conn, r)
	go wsc.run()
}

// pathMatches reports whether any resource's path is compatible with the
// upgrade request, ignoring all other matching criteria.
func (h *PluginHandler) pathMatches(exch *exchange.Exchange) bool {
	for i := range h.config.Resources {
		pathOnly := &config.RequestMatcher{Path: h.config.Resources[i].Path}
		if score, _ := matcher.CalculateMatchScore(exch, pathOnly, nil, h.imposterConfig); score >= 0 {
			return true
		}
	}
	return false
}
