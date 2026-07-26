# Streaming example: Server-Sent Events and scheduled push

Demonstrates incremental HTTP streaming with `stream: true`. Instead of
buffering a single response body, the engine writes and flushes each part to
the client as it is produced:

- **A fixed sequence** — the `responses` list. Each block is flushed
  separately, paced by its `delay`, so an OpenAI-style chat completion arrives
  token-by-token and ends with the `[DONE]` sentinel.
- **Open-ended push** — a `schedule` on a streamed resource pushes further
  events over the open connection on a timer, until its `limit` is reached or
  the client disconnects.

Both are the same core capability the websocket plugin uses to send multiple
and scheduled frames, generalised to HTTP.

## Run

```bash
imposter ./examples/rest/sse-streaming
```

## Try it

A streamed chat completion (use `-N` so curl doesn't buffer):

```bash
curl -N -X POST http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' -d '{"stream":true}'
```

You'll see each `data:` chunk arrive ~50 ms apart, ending with `data: [DONE]`.

An open-ended event stream with a keepalive pushed once a second (stops after
10, or when you press Ctrl-C):

```bash
curl -N http://localhost:8080/events
```

## How it works

`stream: true` opts the resource into incremental delivery. The response is sent
with `Transfer-Encoding: chunked` and each block is flushed as it is written.
Set the `Content-Type` (e.g. `text/event-stream`) on the first block. Write the
SSE framing (`data: ...\n\n`) into the content yourself — the engine streams the
bytes as-is, so the same mechanism also serves NDJSON or any chunked format.

> [!NOTE]
> Streaming needs a long-lived, flushable connection. Under the AWS Lambda
> adapter the blocks are concatenated into a single buffered response instead,
> and schedules (which require an open connection) are skipped.
