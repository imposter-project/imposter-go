# WebSocket example: OpenClaw Gateway simulation

Simulates a subset of the [OpenClaw Gateway WebSocket protocol](https://docs.openclaw.ai/gateway/protocol):

1. On connection, the server immediately sends a `connect.challenge` event and starts a periodic `tick` keepalive every 15 seconds.
2. A `connect` request frame is answered with a `hello-ok` response, echoing the request's `id`.
3. An `agent` request frame is acknowledged with a `res` frame, followed by a stream of `agent` and `chat` events with realistic delays.

The resources use a wildcard path (`path: /*`), so the mock accepts a WebSocket connection on any path — mirroring the real OpenClaw gateway, which multiplexes on a single port and routes the upgrade by the `Upgrade: websocket` header rather than a specific URL path. Connect on whatever path your client uses (e.g. `/ws`).

## Run

```bash
imposter ./examples/websocket/openclaw
```

## Try it

Using [websocat](https://github.com/vi/websocat):

```bash
websocat ws://localhost:8080/ws
```

You'll receive the challenge event immediately. Then paste:

```json
{"type":"req","id":"req-1","method":"connect","params":{}}
```

to receive `hello-ok`, and:

```json
{"type":"req","id":"req-2","method":"agent","params":{"message":"hello"}}
```

to receive an acknowledgement followed by streamed agent/chat events. Leave the connection open to observe the periodic `tick` events.
