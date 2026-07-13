# Schedules example: periodic webhook push

Demonstrates the top-level `schedules` block: while the mock is running, it POSTs a webhook-style notification every 30 seconds, independent of any inbound request. The payload references data captured from earlier requests via a store.

## Run

Start something to receive the webhooks, e.g.:

```bash
python3 -m http.server 9090
```

Then start the mock:

```bash
imposter ./examples/rest/schedules
```

Optionally set `WEBHOOK_URL` to change the destination.

POST an order so the webhook payload has data to reference:

```bash
curl -X POST http://localhost:8080/orders -d '{"id":"order-123"}'
```

Every 30 seconds the mock sends `{"event":"order.updated","orderId":"order-123",...}` to the webhook URL.

Schedules can use `every` (a duration such as `30s` or `5m`) or `cron` (a standard 5-field cron expression) and run any `steps` — `remote` for outbound HTTP, `script` for JavaScript.

The optional `limit` caps how many times a schedule fires; without it, the schedule runs for the lifetime of the mock. Operators can set a global default for schedules that omit `limit` with the `IMPOSTER_SCHEDULE_LIMIT` environment variable.
