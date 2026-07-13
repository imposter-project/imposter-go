# WebSocket support and scheduled push

## Motivation

Imposter's request model was strictly synchronous HTTP request→response. Two roadmap items require the engine to *initiate* traffic:

1. **WebSocket mocking** — e.g. simulating the OpenClaw Gateway API: a client connects, receives a server-initiated challenge event, sends request frames, gets replies, and receives streamed events over time (including a periodic keepalive tick).
2. **Scheduled push** — periodic engine-initiated actions independent of any inbound request, e.g. webhook-style outbound HTTP POSTs on an interval or cron.

Both share a concept — *time-driven, engine-initiated actions* — so they share one `schedules` vocabulary and one runner implementation.

## Config surface

### WebSocket plugin

```yaml
plugin: websocket

resources:
  # Connection opened: handshake matched on path/queryParams/requestHeaders
  - path: /gateway
    on: open                      # open | message (default) | close
    response:
      content: '{"event":"connect.challenge","payload":{"nonce":"${random.uuid()}"}}'
      template: true
    schedule:                     # connection-scoped; stops on close
      - every: 15s
        response:
          content: '{"event":"tick"}'

  # Message received: matched with the standard requestBody matchers
  - path: /gateway
    requestBody:
      jsonPath: $.method
      value: agent
    capture:
      reqId:
        requestBody:
          jsonPath: $.id
    responses:                    # ordered frames; each is a full response block
      - content: '{"type":"res","id":"${stores.request.reqId}","ok":true}'
        template: true
      - file: agent-event.json
        delay: { exact: 250 }
```

Conventions reused: `path`/`queryParams`/`requestHeaders` match the HTTP upgrade request; `requestBody` (jsonPath/xPath, allOf/anyOf, operators) matches each text frame; `capture`, `steps`, `${...}` templating and `delay` behave as elsewhere.

`response` vs `responses`: the singular block remains fully supported everywhere and is exactly equivalent to a `responses` list with one element (`BaseResource.EffectiveResponses()` normalises). Declaring both is a validation error. `responses` is currently restricted to the websocket plugin by validation, but the field lives on `BaseResource` so other plugins can adopt it later.

The request store is **connection-scoped** for websocket configs: one store is shared by the open/message/close events of a connection, so a value captured from an early message (e.g. a request ID) is available to templates on later messages and scheduled frames.

### Schedules

```yaml
plugin: rest
schedules:
  - name: order-webhook
    every: 30s                    # or cron: "0 * * * *"
    steps:
      - type: remote
        url: ${env.WEBHOOK_URL}
        method: POST
        body: '{"event":"order.updated"}'
```

- **Top-level `schedules`** (any plugin): engine-lifetime jobs running `steps` (a `remote` step is an outbound webhook; a `script` step runs JavaScript). Responses are rejected here — there is no client to send to.
- **Resource-level `schedule`** (websocket `on: open` resources only): connection-lifetime jobs that may send `response(s)` frames and/or run `steps`; they stop when the connection closes.
- A schedule entry declares exactly one of `every` (Go duration) or `cron` (standard 5-field expression). Runs are serialised per entry: a firing that outlasts the interval delays the next one rather than overlapping it.
- An optional `limit` caps the total number of firings (omitted = unlimited); once reached, the schedule stops and logs that it has done so. Operators can set a global default via `IMPOSTER_SCHEDULE_LIMIT` for schedules that omit `limit` (an explicit `limit` always wins; no default value is shipped). Docs steer users towards setting a limit for outbound pushes.
- Observability: schedules log registration, trigger and effective limit at INFO (plus a hint when unlimited); each firing, the next fire time, and limit exhaustion at DEBUG/INFO; websocket connections log handled events with frame counts at INFO, unmatched messages at WARN, and frame sends at DEBUG (bodies at TRACE). The resource-level `log:` template is emitted for websocket events, mirroring the HTTP handler.

## Implementation notes

- **In-process plugin** (`plugin/websocket/`), registered in `plugin/plugin.go`. The external plugin RPC contract is unary request/response, so an out-of-process websocket plugin is not viable.
- **Upgrade path**: the handler stores the `http.ResponseWriter` on the `Exchange`; the websocket plugin upgrades via `gorilla/websocket` (connection hijack) and sets `ResponseState.Hijacked`, which makes the handler and `WriteToResponseWriter` skip all writes. Non-hijackable writers (Lambda, RFC 8441 HTTP/2 streams) yield a 501. Under h2c, plain `Upgrade: websocket` requests still arrive on a hijackable HTTP/1.1 writer.
- **Pipeline reuse**: each connection event (open, each text frame, close) runs the standard `pipeline.RunPipeline` with `ProtocolHooks`: the score hook filters resources by `on` (defaulting to `message`) before the standard matcher; the response hook runs standard response processing (delay, file/content, templating) and enqueues the body as a text frame. Multi-frame streaming falls out of the pipeline iterating `EffectiveResponses()`. Frames from `on: close` resources are suppressed (steps/captures still run).
- **Connection concurrency**: one writer goroutine per connection drains a bounded channel (gorilla forbids concurrent writers); producers are the read-loop replies and schedule firings. A context cancelled on close stops schedules and the writer. Scheduled frames may interleave between a resource's multi-frame sequence; frame atomicity is guaranteed, sequence atomicity is not.
- **Scheduler** (`internal/scheduler/`): a shared runner (`TriggerFunc` + `RunSchedule`) drives both engine-lifetime schedules (started by the HTTP adapter after initialisation, stopped in `cleanup()`) and per-connection schedules. `robfig/cron/v3` is used only for parsing/next-time computation.
- **Lambda**: schedules and websocket configs log a startup warning; neither can work in that execution model.

## Limitations (MVP)

- Text frames only; binary frames are logged and ignored.
- No ping/pong keepalive or read deadlines (use a scheduled application-level tick if needed).
- No RFC 8441 (WebSocket over HTTP/2).
- Interceptors run per connection event, filtered by `on` like resources.
