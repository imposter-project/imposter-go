# Simple REST Example

This example demonstrates basic REST API mocking capabilities of Imposter, including:
- Path matching
- HTTP method matching
- Response configuration
- Response headers
- Response status codes

## Configuration

The example uses a simple configuration file (`imposter-config.yaml`) that defines several REST endpoints:

```yaml
plugin: rest
resources:
  - path: /hello
    method: GET
    response:
      content: Hello, World!
      statusCode: 200
      headers:
        Content-Type: text/plain

  - path: /users/{id}
    method: GET
    pathParams:
      id:
        value: "123"
    response:
      content: |
        {
          "id": 123,
          "name": "John Doe",
          "email": "john@example.com"
        }
      statusCode: 200
      headers:
        Content-Type: application/json

  - path: /users
    method: POST
    requestBody:
      jsonPath: "$.name"
      value: "John Doe"
    response:
      content: |
        {
          "id": 456,
          "name": "John Doe",
          "email": "john@example.com"
        }
      statusCode: 201
      headers:
        Content-Type: application/json
        Location: /users/456
```

## Testing the Example

You can test the example using curl commands:

1. Get a simple greeting:
```bash
curl http://localhost:8080/hello
```
Expected response: `Hello, World!`

2. Get a user by ID:
```bash
curl http://localhost:8080/users/123
```
Expected response:
```json
{
  "id": 123,
  "name": "John Doe",
  "email": "john@example.com"
}
```

3. Create a new user:
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}' \
  http://localhost:8080/users
```
Expected response:
```json
{
  "id": 456,
  "name": "John Doe",
  "email": "john@example.com"
}
```
The response will include a `Location` header pointing to `/users/456` and a status code of 201 (Created).

## Features Demonstrated

1. **Path Matching**: Different URL patterns including static paths (`/hello`) and paths with parameters (`/users/{id}`).

2. **HTTP Methods**: Support for different HTTP methods (GET, POST).

3. **Path Parameters**: Matching specific path parameter values (e.g., `id: "123"`).

4. **Request Body Matching**: Using JSONPath to match specific fields in JSON request bodies.

5. **Response Configuration**:
   - Custom status codes (200, 201)
   - Custom headers (Content-Type, Location)
   - Different response content types (text/plain, application/json)
   - JSON responses with proper formatting

6. **Content Types**: Handling both plain text and JSON responses with appropriate Content-Type headers. 