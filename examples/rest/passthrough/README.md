# Passthrough / Upstream Proxy Example

Demonstrates the partial-mocking pattern: mock a subset of endpoints locally
and forward the rest to a real upstream service.

## How it works

`upstreams` defines a named backend (`jsonplaceholder`). Resources can either
serve a mocked response as usual, or set `passthrough: <upstream-name>` to
forward the matched request to that upstream and return its response verbatim.

When a request matches both an exact-path resource and a wildcard one, the
exact path wins — so `/users/1` returns the local fixture while `/users/2`,
`/users/3` etc. fall through to the wildcard and get proxied.

See [docs/passthrough.md](../../../docs/passthrough.md) for the full feature
reference.

## Running

From this directory:

```bash
imposter -d .
```

The example requires outbound network access to `https://jsonplaceholder.typicode.com`.

## Testing

`/users/1` is served from the local mock:

```bash
curl http://localhost:8080/users/1
# {
#   "id": 1,
#   "name": "Locally Mocked User",
#   "email": "mock@example.com"
# }
```

Any other user ID is proxied to the upstream:

```bash
curl http://localhost:8080/users/2
# {"id": 2, "name": "Ervin Howell", ...}   ← from jsonplaceholder.typicode.com
```

Posts are proxied wholesale:

```bash
curl http://localhost:8080/posts/1
# {"userId": 1, "id": 1, "title": "...", "body": "..."}
```

## Features demonstrated

1. **Named upstreams** — top-level `upstreams` map declaring backend URLs
2. **Per-resource opt-in** — a resource forwards by setting
   `passthrough: <name>`; without it, the resource serves a mocked response
3. **Partial mocking** — exact-path mock plus a wildcard passthrough lets you
   override individual endpoints while proxying the rest
4. **Verbatim forwarding** — the upstream's status, headers and body are
   returned to the client; `response`, `steps` and templating are bypassed for
   passthrough resources

## Environment variables

- `IMPOSTER_PASSTHROUGH_TIMEOUT` — override the 30s upstream timeout
- `IMPOSTER_PASSTHROUGH_FORWARDED_HEADERS=true` — inject `X-Forwarded-*` and
  `Via` headers on proxied requests
