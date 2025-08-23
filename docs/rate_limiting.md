# Rate Limiting

Rate limiting in Imposter allows you to control the number of concurrent requests to your mock endpoints. This is useful for simulating real-world API behavior, testing system resilience, and preventing overload during load testing.

## Overview

The rate limiter tracks active concurrent requests per resource and applies different responses when configurable thresholds are exceeded. It supports:

- **Concurrent request limiting**: Control how many requests can be processed simultaneously
- **Multiple threshold tiers**: Define different responses for different concurrency levels
- **Progressive throttling**: Add delays, return errors, or apply custom responses
- **Store backend flexibility**: Works with in-memory, Redis, and DynamoDB stores
- **Distributed operation**: Consistent behavior across multiple server instances (with shared stores)

## Configuration

Rate limiting is configured per resource using the `concurrency` property. Each concurrency limit defines a threshold and the response to return when that threshold is exceeded.

### Basic Rate Limiting

```yaml
plugin: rest
resources:
  - path: /api/users
    method: GET
    concurrency:
      - threshold: 5
        response:
          statusCode: 429
          content: "Too many concurrent requests"
          headers:
            Retry-After: "10"
    response:
      statusCode: 200
      content: "User data"
```

In this example:
- Normal requests (1-5 concurrent) return the standard 200 response
- When more than 5 requests are active, additional requests get a 429 "Too Many Requests" response

### Multiple Thresholds

You can define multiple concurrency thresholds with different responses for progressive rate limiting:

```yaml
plugin: rest
resources:
  - path: /api/heavy-operation
    method: POST
    concurrency:
      # First tier: Add delay to slow down requests
      - threshold: 3
        response:
          statusCode: 200
          content: "Request throttled"
          delay:
            exact: 2000  # 2 second delay
          headers:
            X-Throttled: "true"
      
      # Second tier: Return 503 Service Unavailable
      - threshold: 7
        response:
          statusCode: 503
          content: "Service temporarily overloaded"
          headers:
            Retry-After: "30"
      
      # Third tier: Return 429 Rate Limit Exceeded
      - threshold: 10
        response:
          statusCode: 429
          content: "Rate limit exceeded"
    response:
      statusCode: 202
      content: "Operation accepted"
```

With this configuration:
- **1-3 concurrent requests**: Normal 202 response
- **4-7 concurrent requests**: 200 response with 2-second delay and throttling header
- **8-10 concurrent requests**: 503 Service Unavailable
- **11+ concurrent requests**: 429 Rate Limit Exceeded

### SOAP Rate Limiting

Rate limiting works with SOAP endpoints using the operation name:

```yaml
plugin: soap
resources:
  - operation: getUserDetails
    concurrency:
      - threshold: 3
        response:
          statusCode: 503
          content: |
            <soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
              <soap:Body>
                <soap:Fault>
                  <faultcode>Server</faultcode>
                  <faultstring>Service temporarily overloaded</faultstring>
                </soap:Fault>
              </soap:Body>
            </soap:Envelope>
    response:
      statusCode: 200
      content: |
        <soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
          <soap:Body>
            <getUserDetailsResponse>
              <user>
                <id>123</id>
                <name>John Doe</name>
              </user>
            </getUserDetailsResponse>
          </soap:Body>
        </soap:Envelope>
```

## Response Configuration

Rate limit responses support all the same features as regular responses:

### Status Codes and Content
```yaml
concurrency:
  - threshold: 5
    response:
      statusCode: 429
      content: "Rate limit exceeded"
```

### Custom Headers
```yaml
concurrency:
  - threshold: 5
    response:
      statusCode: 503
      headers:
        Retry-After: "30"
        X-RateLimit-Limit: "5"
        X-RateLimit-Remaining: "0"
      content: "Service temporarily unavailable"
```

### Delays
```yaml
concurrency:
  - threshold: 3
    response:
      delay:
        exact: 1000  # Fixed 1-second delay
      statusCode: 200
      content: "Throttled response"

  - threshold: 5
    response:
      delay: # Random delay between 0.5-2 seconds
        min: 500
        max: 2000
      statusCode: 200
      content: "Heavily throttled"
```

### Templated Responses
```yaml
concurrency:
  - threshold: 5
    response:
      statusCode: 429
      template: true
      content: |
        {
          "error": "Rate limit exceeded",
          "timestamp": "${datetime.now.iso8601_datetime}",
          "retry_after": 30
        }
      headers:
        Content-Type: application/json
```

## Store Backends

Rate limiting works with different store backends for different deployment scenarios:

### In-Memory Store (Default)
```bash
# No configuration needed - this is the default
imposter examples/rest/rate-limiting
```

**Characteristics:**
- Fast performance
- Per-instance rate limiting
- Suitable for single-instance deployments
- Data lost on restart

### Redis Store
```bash
export IMPOSTER_STORE_DRIVER=store-redis
export REDIS_ADDR=localhost:6379
imposter examples/rest/rate-limiting
```

**Characteristics:**
- Shared rate limiting across multiple instances
- Persistent across restarts
- High performance with atomic operations
- Suitable for distributed deployments

### DynamoDB Store
```bash
export IMPOSTER_STORE_DRIVER=store-dynamodb
export IMPOSTER_DYNAMODB_TABLE=imposter-store
imposter examples/rest/rate-limiting
```

**Characteristics:**
- Shared rate limiting across multiple instances
- Fully managed and scalable
- Automatic TTL support
- Suitable for cloud deployments

## Environment Variables

Configure rate limiting behavior with these environment variables:

| Variable | Purpose | Default | Example |
|----------|---------|---------|---------|
| `IMPOSTER_RATE_LIMITER_TTL` | TTL for rate limit entries (seconds) | `300` (5 minutes) | `IMPOSTER_RATE_LIMITER_TTL=600` |
| `IMPOSTER_STORE_DRIVER` | Store backend selection | `inmemory` | `IMPOSTER_STORE_DRIVER=store-redis` |
| `REDIS_ADDR` | Redis server address (if using Redis) | - | `REDIS_ADDR=localhost:6379` |
| `IMPOSTER_DYNAMODB_TABLE` | DynamoDB table name (if using DynamoDB) | - | `IMPOSTER_DYNAMODB_TABLE=imposter-store` |

## How It Works

### Threshold Logic
Rate limiting uses "greater than" logic to determine when thresholds are exceeded:

- **Threshold: 3** means rate limiting applies when there are **more than 3** concurrent requests (i.e., 4 or more)
- Multiple thresholds are evaluated in order, with the highest matching threshold taking precedence

### Request Processing Flow
1. **Request arrives** at the configured endpoint
2. **Counter incremented** atomically for the resource
3. **Threshold evaluation** checks if any limits are exceeded
4. **If rate limited**: Counter is rolled back and rate limit response is returned
5. **If not rate limited**: Request processes normally
6. **After response**: Counter is decremented automatically

### Resource Identification
Each resource is identified by a unique key that includes:
- **HTTP method** (GET, POST, etc.) or "*" for SOAP
- **Resource path** (REST) or operation name (SOAP)  
- **Hash of matching criteria** (if the resource has specific headers, query params, etc.)

This ensures that resources with the same path but different matching criteria get separate rate limit counters.

## Error Handling

The rate limiter follows a "fail-open" approach:
- **Store failures** allow requests to proceed normally (rate limiting is disabled)
- **Network issues** don't block request processing
- **Counter errors** are logged but don't impact request flow

This ensures that rate limiting enhances your testing without compromising availability.

## Load Testing

Rate limiting integrates well with load testing tools like `hey`:

```bash
# Install hey
brew install hey  # macOS
go install github.com/rakyll/hey@latest  # Other platforms

# Test concurrent requests
hey -n 100 -c 20 -m GET http://localhost:8080/api/users

# Test different endpoints
hey -n 50 -c 10 -m POST http://localhost:8080/api/heavy-operation
```

Monitor the output for status code distribution to see rate limiting in action:
```
Status code distribution:
  [200]  75 responses  # Normal responses
  [429]  20 responses  # Rate limited
  [503]   5 responses  # Service overloaded
```

## Best Practices

### 1. Progressive Throttling
Use multiple thresholds to handle load gracefully:
```yaml
concurrency:
  - threshold: 5    # Warn with delays
    response:
      delay: { exact: 1000 }
      headers: { X-Throttled: "true" }
  - threshold: 10   # Soft rejection
    response:
      statusCode: 503
      headers: { Retry-After: "10" }
  - threshold: 20   # Hard rejection
    response:
      statusCode: 429
```

### 2. Appropriate HTTP Status Codes
- **429 Too Many Requests**: For hard rate limits
- **503 Service Unavailable**: For temporary overload
- **200 OK with delays**: For throttling without errors

### 3. Helpful Headers
Include headers to help clients understand the rate limiting:
```yaml
headers:
  Retry-After: "30"
  X-RateLimit-Limit: "10"
  X-RateLimit-Remaining: "0"
  X-RateLimit-Reset: "${datetime.now.plus_seconds(30).epoch_second}"
```

### 4. Realistic Thresholds
Set thresholds based on your actual API capacity:
- **Database APIs**: Lower thresholds (2-5 concurrent)
- **Cache APIs**: Higher thresholds (20-50 concurrent)
- **File uploads**: Very low thresholds (1-2 concurrent)

### 5. Store Selection
Choose the appropriate store backend for your testing scenario:
- **In-memory**: Single instance testing
- **Redis**: Multi-instance testing with shared state
- **DynamoDB**: Cloud-based distributed testing
