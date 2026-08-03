package emit

import (
	"net/http"
	"sync"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/imposter-project/imposter-go/internal/exchange"
	"github.com/imposter-project/imposter-go/internal/response"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// httpSink writes processed response bodies to an HTTP client incrementally,
// flushing after each so the client receives them as they are produced. The
// status line and headers are written once, from the first emitted response.
type httpSink struct {
	w           http.ResponseWriter
	flusher     http.Flusher
	req         *http.Request
	wroteHeader bool
}

// Emit writes and flushes one response body, returning false once the client
// has disconnected or a write fails so the caller stops.
func (s *httpSink) Emit(rs *exchange.ResponseState) bool {
	if s.req.Context().Err() != nil {
		// Client has gone away.
		return false
	}
	if !s.wroteHeader {
		for key, value := range rs.Headers {
			s.w.Header().Set(key, value)
		}
		status := rs.StatusCode
		if status == 0 {
			status = http.StatusOK
		}
		s.w.WriteHeader(status)
		s.wroteHeader = true
	}
	if len(rs.Body) > 0 {
		if _, err := s.w.Write(rs.Body); err != nil {
			logger.Debugf("stream write failed - path:%s, error:%v", s.req.URL.Path, err)
			return false
		}
	}
	s.flusher.Flush()
	return true
}

// StreamHTTP writes a matched resource's responses to the HTTP client
// incrementally, flushing each, then runs any request-scoped schedules until
// they reach their limit or the client disconnects. It marks the response
// state Streamed so the buffered writer does not write the body again.
//
// When the underlying writer cannot flush (e.g. the AWS Lambda adapter), it
// falls back to buffering every response into a single body written the normal
// way, and skips schedules (which need a live, flushable connection).
func StreamHTTP(
	exch *exchange.Exchange,
	resource *config.BaseResource,
	respProc response.Processor,
	imposterConfig *config.ImposterConfig,
	configDir string,
) {
	rs := exch.ResponseState
	reqMatcher := &resource.RequestMatcher
	resps := resource.EffectiveResponses()

	flusher, ok := exch.ResponseWriter.(http.Flusher)
	if !ok {
		streamBuffered(exch, resource, resps, respProc)
		return
	}

	req := exch.Request.Request
	logger.Infof("streaming response - method:%s, path:%s, chunks:%d, schedules:%d",
		req.Method, req.URL.Path, len(resps), len(resource.Schedule))

	sink := &httpSink{w: exch.ResponseWriter, flusher: flusher, req: req}

	// Emit the fixed sequence first (e.g. an SSE body ending in [DONE]).
	EmitResponses(exch, reqMatcher, resps, respProc, sink)

	// Then run any request-scoped schedules (open-ended server push). This
	// blocks until every schedule hits its limit or the client disconnects,
	// keeping the HTTP response open in the meantime.
	if len(resource.Schedule) > 0 {
		var wg sync.WaitGroup
		host := &ScheduleHost{
			Ctx:            req.Context(),
			Schedules:      resource.Schedule,
			Label:          "request " + req.URL.Path,
			ImposterConfig: imposterConfig,
			ConfigDir:      configDir,
			RespProc:       respProc,
			Sink:           sink,
			NewExchange: func() *exchange.Exchange {
				return exchange.NewExchange(req, nil, exch.RequestStore, response.NewResponseState())
			},
		}
		host.Start(&wg)
		wg.Wait()
	}

	rs.Streamed = true
	rs.HandledWithResource(resource)
}

// streamBuffered is the non-flushing fallback: it concatenates every response
// body into one, written via the normal buffered path, and warns that
// schedules are unsupported without a flushable writer.
func streamBuffered(
	exch *exchange.Exchange,
	resource *config.BaseResource,
	resps []config.Response,
	respProc response.Processor,
) {
	rs := exch.ResponseState
	req := exch.Request.Request
	logger.Debugf("streaming unavailable (writer is not flushable); buffering %d response(s) - path:%s", len(resps), req.URL.Path)

	var buf []byte
	for i := range resps {
		respProc(exch, &resource.RequestMatcher, &resps[i])
		buf = append(buf, rs.Body...)
		rs.Body = nil
		rs.File = ""
	}
	rs.Body = buf

	if len(resource.Schedule) > 0 {
		logger.Warnf("resource (path %q) declares a schedule but the connection cannot stream; schedule skipped", resource.Path)
	}
}
