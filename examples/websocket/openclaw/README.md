# WebSocket example: OpenClaw Gateway simulation

Simulates a subset of the [OpenClaw Gateway WebSocket protocol](https://docs.openclaw.ai/gateway/protocol) — enough for an operator client (e.g. the lucinate TUI) to connect, browse a single agent and session, and hold a simple chat:

1. On connection, the server immediately sends a `connect.challenge` event and starts a periodic `tick` keepalive every 15 seconds. The `ts`/`timestamp` fields are numbers (epoch millis), matching the protocol's `int64` typing.
2. A `connect` request frame is answered with a `hello-ok` response (protocol 4), echoing the request's `id`. The hello snapshot advertises a `main` agent and its `main` session.
3. An `agents.list` request returns a single agent, `main` (also the default).
4. A `sessions.list` request returns one `main` session.
5. A `cron.list` request returns no jobs.
6. A `sessions.create` request returns the `main` session key (there is a single session, so create/resume always lands on `main`).
7. A `chat.history` request returns a short seed conversation, so the chat screen has something to render. Return `{"messages":[]}` for an empty history.
8. A `chat.send` request is acknowledged with a `res` frame (`runId`/`status`), followed by streamed `agent` (tool start/result) and `chat` (`delta` then `final`) events with realistic delays. The `runId` is the request id and the `sessionKey` is echoed from the request, so the client routes the events to the active run.

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

You'll receive the challenge event immediately. Then paste a handshake:

```json
{"type":"req","id":"req-1","method":"connect","params":{"minProtocol":3,"maxProtocol":4,"client":{"id":"cli","version":"0.1.0","platform":"go","mode":"cli"}}}
```

to receive `hello-ok`. List the agent, session and cron jobs:

```json
{"type":"req","id":"req-2","method":"agents.list","params":{}}
{"type":"req","id":"req-3","method":"sessions.list","params":{"agentId":"main"}}
{"type":"req","id":"req-4","method":"cron.list","params":{"enabled":"all"}}
```

open the session and load its history:

```json
{"type":"req","id":"req-5","method":"sessions.create","params":{"agentId":"main","key":"main"}}
{"type":"req","id":"req-6","method":"chat.history","params":{"sessionKey":"main","limit":50}}
```

and send a chat message:

```json
{"type":"req","id":"req-7","method":"chat.send","params":{"sessionKey":"main","message":"hello","idempotencyKey":"idem-1"}}
```

to receive an acknowledgement followed by streamed tool and chat events. Leave the connection open to observe the periodic `tick` events.
