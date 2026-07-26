// Package emit holds the transport-agnostic machinery for sending one or more
// processed responses to a client over a long-lived exchange — a websocket
// connection or a streamed HTTP response. The websocket plugin and the HTTP
// streaming path share these primitives so that "process a response block,
// then send it" (and "fire a schedule that emits responses") lives in one
// place rather than being duplicated per protocol.
package emit

import (
	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
)

// Sink delivers a processed response body to a client transport (a websocket
// text frame, an HTTP response stream, ...). It decouples the shared response
// handling from the protocol that carries the bytes.
type Sink interface {
	// Emit sends the body currently held in the response state. It returns
	// false when the sink can accept no more — typically because the client
	// has gone away — signalling callers to stop emitting further responses.
	Emit(rs *exchange.ResponseState) bool
}

// DiscardSink accepts and drops every body. It is used where responses must be
// processed for their side effects (captures, steps, templating) but must not
// be sent — for example a websocket 'close' event, after which no frame can be
// delivered.
type DiscardSink struct{}

// Emit implements Sink by discarding the body.
func (DiscardSink) Emit(*exchange.ResponseState) bool { return true }

// EmitOne sends the body currently in the response state via the sink, if any,
// then clears the per-response fields so the next response starts clean. It
// reports whether the sink accepted the body (false means stop). A response
// with an empty body is a no-op that always reports true.
func EmitOne(rs *exchange.ResponseState, sink Sink) bool {
	accepted := true
	if len(rs.Body) > 0 {
		accepted = sink.Emit(rs)
	}
	rs.Body = nil
	rs.File = ""
	return accepted
}

// EmitResponses processes each response block through respProc and emits the
// result via the sink, in order, stopping early if the sink stops accepting
// (client gone). Per-block delays configured on each response are honoured by
// respProc before the block is emitted, which paces the stream. It returns the
// number of non-empty bodies emitted.
func EmitResponses(
	exch *exchange.Exchange,
	reqMatcher *config.RequestMatcher,
	resps []config.Response,
	respProc response.Processor,
	sink Sink,
) int {
	rs := exch.ResponseState
	var emitted int
	for i := range resps {
		respProc(exch, reqMatcher, &resps[i])
		hadBody := len(rs.Body) > 0
		accepted := EmitOne(rs, sink)
		if hadBody && accepted {
			emitted++
		}
		if !accepted {
			break
		}
	}
	return emitted
}
