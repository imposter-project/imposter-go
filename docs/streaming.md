# Streaming responses

By default a resource produces a single response that is buffered and written in
one go. Setting `stream: true` on a resource instead delivers its output
**incrementally**: each part is written and flushed to the client as it is
produced, over a chunked (`Transfer-Encoding: chunked`) HTTP response.

This drives Server-Sent Events (SSE), newline-delimited JSON (NDJSON), and any
other chunked format.

Streaming is supported by the plugins that serve responses through the shared
request pipeline: `rest` and `openapi`.

## A fixed sequence of chunks

List the parts under `responses`. Each is processed like a normal response
(`file`/`content`, `template`, `headers`, `statusCode`) and flushed separately.
A per-part `delay` paces the stream.

```yaml
plugin: rest
resources:
  - path: /updates
    method: GET
    stream: true
    responses:
      - content: "data: working\n\n"
        headers:
          Content-Type: text/event-stream
        delay: { exact: 500 }
      - content: "data: still working\n\n"
        delay: { exact: 500 }
      - content: "data: done\n\n"
```

A client reading this receives three `data:` events half a second apart, rather
than all at once at the end.

The status line and headers are taken from the first part, so set the
`Content-Type` (e.g. `text/event-stream`) there. Write the SSE framing
(`data: ...\n\n`) into the content yourself — the engine streams the bytes
as-is.

`responses` (the plural, multi-part form) requires `stream: true`. A single
`response` block is the normal, buffered form unless you also set `stream: true`.

## Open-ended server push

A `schedule` on a streamed resource keeps the connection open and pushes further
responses on a timer — an SSE keepalive, progress updates, live data. It reuses
the same schedule triggers as [top-level and websocket schedules](#see-also)
(`every` or `cron`, with an optional `limit`).

```yaml
plugin: rest
resources:
  - path: /events
    method: GET
    stream: true
    response:
      content: ": stream open\n\n"
      headers:
        Content-Type: text/event-stream
    schedule:
      - name: keepalive
        every: 1s
        limit: 10          # omit to push for as long as the client stays connected
        response:
          template: true
          content: "data: {\"type\":\"ping\",\"at\":\"${datetime.now.iso8601_datetime}\"}\n\n"
```

The request handler holds the response open while the schedules run. It returns
— ending the response — once every schedule reaches its `limit`, or as soon as
the client disconnects. As with all schedules, set a `limit` (or the global
`IMPOSTER_SCHEDULE_LIMIT`) unless the stream genuinely should run for as long as
the client is connected.

## Notes and limits

- **Flushable connection required.** Streaming needs a long-lived, flushable
  HTTP connection. Under the AWS Lambda adapter — which buffers a single
  response — the parts are concatenated into one body instead, and schedules
  (which need an open connection) are skipped with a warning.
- **Content type inference.** As with any response, if no `Content-Type` is set
  it is inferred from the response file extension, falling back to JSON.
- **Websocket is always streaming.** A websocket connection is inherently a
  multi-frame stream, so `stream` is implicit there: setting `stream: true` is
  redundant but accepted, while an explicit `stream: false` is rejected at
  startup, since the plugin cannot honour it.

## See also

- [Rate limiting](./rate_limiting.md)
- The `examples/rest/sse-streaming` example
- The websocket plugin, whose multiple-response and scheduled-frame sending
  shares the same core (`internal/emit`).
