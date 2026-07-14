# WebSocket example: echo

A minimal WebSocket mock — the "hello world" of the `websocket` plugin:

1. On connect, the server sends a greeting.
2. Each message received is echoed back, using `${context.request.body}` in a templated response.

The resources use a wildcard path (`path: /*`), so the mock accepts a connection on any path.

For a richer example — connection-lifecycle events, request/response matching, streamed replies and periodic pushes — see the [openclaw](../openclaw) example.

## Run

```bash
imposter ./examples/websocket/echo
```

## Try it

Using [websocat](https://github.com/vi/websocat):

```bash
websocat ws://localhost:8080/
```

You'll receive the greeting immediately. Then type any message and it will be echoed back as `You said: <your message>`.
