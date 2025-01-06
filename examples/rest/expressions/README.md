# Expression Matching Example

This example demonstrates how to use expression matching with `allOf` and `anyOf` conditions in Imposter.

## Configuration

The example includes two endpoints:

1. `/api/users/{id}` - Demonstrates `anyOf` matching:
   - Matches if either:
     - The `Authorization` header equals "Bearer admin-token" OR
     - The `apiKey` query parameter equals "secret-key"

2. `/api/orders` - Demonstrates `allOf` matching:
   - Matches only if both:
     - The `X-User-Role` header equals "admin" AND
     - The `region` query parameter equals "EU"

## Testing

You can test the endpoints using curl:

### Testing anyOf matching

The `/api/users/{id}` endpoint will match if either condition is met:

Using the Authorization header:
```bash
curl -H "Authorization: Bearer admin-token" http://localhost:8080/api/users/123
```

Using the API key:
```bash
curl "http://localhost:8080/api/users/123?apiKey=secret-key"
```

### Testing allOf matching

The `/api/orders` endpoint requires both conditions to be met:

```bash
curl -H "X-User-Role: admin" "http://localhost:8080/api/orders?region=EU"
```

## Expression Operators

The following operators are available for expression matching:

- `EqualTo` (default) - Exact string match
- `NotEqualTo` - String does not match
- `Contains` - String contains the value
- `NotContains` - String does not contain the value
- `Matches` - Regular expression match
- `NotMatches` - Regular expression does not match
- `Exists` - Value is not empty
- `NotExists` - Value is empty

## Expression Context

Expressions can access various parts of the request context:

- `${context.request.headers.HEADER_NAME}` - Request headers
- `${context.request.queryParams.PARAM_NAME}` - Query parameters
- `${context.request.pathParams.PARAM_NAME}` - Path parameters
- `${stores.STORE_NAME.KEY}` - Store values 