# Rate Limiting Example

This example demonstrates the concurrent request rate limiting capabilities of Imposter, including:
- Per-resource concurrency limits
- Multiple rate limiting tiers with different responses
- Delay injection for throttling
- Error responses when limits are exceeded
- Load testing with the `hey` tool

## Overview

The rate limiter works by tracking concurrent requests per resource across multiple server instances. When a limit is exceeded, it returns a configured response instead of processing the request normally. This is particularly useful for:

- **API throttling**: Prevent overload of backend services
- **Load testing**: Simulate realistic server behavior under load
- **Performance testing**: Introduce controlled delays and failures
- **Capacity planning**: Test system behavior at various concurrency levels

## Configuration

The example uses a configuration file (`imposter-config.yaml`) that defines REST endpoints with different concurrency limits:

```yaml
plugin: rest
resources:
  # Light endpoint - can handle 10 concurrent requests
  - path: /api/light
    method: GET
    concurrency:
      - limit: 10
        response:
          statusCode: 429
          content: "Too many concurrent requests to light endpoint"
          headers:
            Retry-After: "5"
    response:
      template: true
      content: |
        {
          "message": "Light endpoint response",
          "timestamp": "${datetime.now.iso8601_datetime}",
          "endpoint": "light"
        }
      headers:
        Content-Type: application/json

  # Heavy endpoint - lower concurrency limit with progressive throttling
  - path: /api/heavy
    method: GET
    concurrency:
      # First tier: add delay to slow down requests
      - limit: 3
        response:
          template: true
          content: |
            {
              "message": "Heavy endpoint - throttled",
              "timestamp": "${datetime.now.iso8601_datetime}",
              "endpoint": "heavy",
              "status": "throttled"
            }
          delay:
            exact: 2000
          headers:
            Content-Type: application/json
            X-Throttled: "true"
      
      # Second tier: reject with 503 Service Unavailable
      - limit: 5
        response:
          statusCode: 503
          template: true
          content: |
            {
              "error": "Service temporarily overloaded",
              "message": "Heavy endpoint is currently overloaded. Please try again later.",
              "timestamp": "${datetime.now.iso8601_datetime}",
              "retryAfter": 30
            }
          headers:
            Content-Type: application/json
            Retry-After: "30"
    response:
      template: true
      content: |
        {
          "message": "Heavy endpoint response",
          "timestamp": "${datetime.now.iso8601_datetime}",
          "endpoint": "heavy",
          "processingTime": "normal"
        }
      delay:
        range:
          min: 100
          max: 500
      headers:
        Content-Type: application/json

  # Critical endpoint - very strict limits
  - path: /api/critical
    method: POST
    concurrency:
      - limit: 2
        response:
          statusCode: 429
          template: true
          content: |
            {
              "error": "Rate limit exceeded",
              "message": "Critical endpoint allows maximum 2 concurrent requests",
              "timestamp": "${datetime.now.iso8601_datetime}",
              "endpoint": "critical"
            }
          headers:
            Content-Type: application/json
            Retry-After: "10"
    response:
      template: true
      content: |
        {
          "message": "Critical operation completed",
          "timestamp": "${datetime.now.iso8601_datetime}",
          "endpoint": "critical",
          "status": "success"
        }
      delay:
        exact: 1000
      headers:
        Content-Type: application/json

  # Status endpoint - no rate limiting for monitoring
  - path: /api/status
    method: GET
    response:
      template: true
      content: |
        {
          "status": "healthy",
          "timestamp": "${datetime.now.iso8601_datetime}",
          "version": "1.0.0",
          "endpoints": {
            "light": "up to 10 concurrent",
            "heavy": "up to 3 concurrent (throttled), up to 5 (rejected)",
            "critical": "up to 2 concurrent"
          }
        }
      headers:
        Content-Type: application/json
```

## Running the Example

1. **Start the Imposter server**:
   ```bash
   # From the project root
   go run cmd/imposter/main.go -configDir examples/rest/rate-limiting
   ```

2. **Test basic functionality**:
   ```bash
   # Test the status endpoint (no rate limiting)
   curl http://localhost:8080/api/status | jq

   # Test light endpoint (allows 10 concurrent)
   curl http://localhost:8080/api/light | jq

   # Test heavy endpoint (throttles at 3, rejects at 5)
   curl http://localhost:8080/api/heavy | jq

   # Test critical endpoint (rejects at 2)
   curl -X POST http://localhost:8080/api/critical | jq
   ```

## Load Testing with `hey`

The `hey` tool is perfect for testing rate limiting behavior. Install it first:

```bash
# macOS
brew install hey

# Linux
go install github.com/rakyll/hey@latest

# Or download from: https://github.com/rakyll/hey/releases
```

### Test Scenarios

#### 1. Light Endpoint Load Test
Test the light endpoint with 20 concurrent requests:

```bash
# Send 100 requests with 20 concurrent connections
hey -n 100 -c 20 -m GET http://localhost:8080/api/light
```

Expected behavior:
- Most requests should succeed (200 OK)
- Some requests may be rate limited (429) when >10 concurrent
- Response times should be consistent

#### 2. Heavy Endpoint Progressive Load Test
Test the heavy endpoint's progressive throttling:

```bash
# Test with 10 concurrent requests to trigger both throttling tiers
hey -n 50 -c 10 -m GET http://localhost:8080/api/heavy
```

Expected behavior:
- First 3 concurrent: Normal response with delay (200 OK)
- Next 2 concurrent: Throttled response with 2s delay (200 OK)
- Beyond 5 concurrent: Service unavailable (503)

#### 3. Critical Endpoint Stress Test
Test the critical endpoint's strict limits:

```bash
# Test with 5 concurrent POST requests
hey -n 20 -c 5 -m POST -H "Content-Type: application/json" \
    -d '{"operation": "critical_task"}' \
    http://localhost:8080/api/critical
```

Expected behavior:
- Only 2 concurrent requests allowed
- Remaining requests get 429 (Too Many Requests)
- High rate of rate limiting

#### 4. Mixed Load Test
Test multiple endpoints simultaneously:

```bash
# Terminal 1: Light load on light endpoint
hey -n 200 -c 5 -m GET http://localhost:8080/api/light &

# Terminal 2: Heavy load on heavy endpoint  
hey -n 100 -c 8 -m GET http://localhost:8080/api/heavy &

# Terminal 3: Burst on critical endpoint
hey -n 30 -c 10 -m POST http://localhost:8080/api/critical &

# Wait for all to complete
wait
```

### Analyzing Results

The `hey` output provides valuable metrics:

```
Summary:
  Total:	2.1234 secs
  Slowest:	2.5678 secs
  Fastest:	0.1234 secs
  Average:	0.8765 secs
  Requests/sec:	47.12
  
Status code distribution:
  [200]	85 responses
  [429]	15 responses
  [503]	5 responses
```

Key metrics to observe:
- **Status code distribution**: Shows how many requests were rate limited
- **Response times**: Throttled requests will show longer times
- **Requests/sec**: Overall throughput considering rate limiting

## Environment Variables

Configure rate limiter behavior with environment variables:

```bash
# Set TTL for rate limiter entries (default: 300 seconds)
export IMPOSTER_RATE_LIMITER_TTL=300

# Use different store backends
export IMPOSTER_STORE_DRIVER=store-redis    # For Redis
export IMPOSTER_STORE_DRIVER=store-dynamodb # For DynamoDB
# Default is in-memory store

# Redis configuration (if using Redis store)
export REDIS_ADDR=localhost:6379

# DynamoDB configuration (if using DynamoDB store)
export IMPOSTER_DYNAMODB_TABLE=imposter-store
```

## Testing Different Store Backends

### In-Memory Store (Default)
```bash
# No environment variables needed
go run cmd/imposter/main.go -configDir examples/rest/rate-limiting
```

### Redis Store
```bash
# Start Redis (if not running)
docker run -d -p 6379:6379 redis:alpine

# Run with Redis store
export IMPOSTER_STORE_DRIVER=store-redis
export REDIS_ADDR=localhost:6379
go run cmd/imposter/main.go -configDir examples/rest/rate-limiting
```

### DynamoDB Store
```bash
# Start local DynamoDB (for testing)
docker run -d -p 8000:8000 amazon/dynamodb-local

# Create table (one-time setup)
aws dynamodb create-table \
  --table-name imposter-store \
  --attribute-definitions AttributeName=StoreName,AttributeType=S AttributeName=Key,AttributeType=S \
  --key-schema AttributeName=StoreName,KeyType=HASH AttributeName=Key,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --endpoint-url http://localhost:8000

# Run with DynamoDB store
export IMPOSTER_STORE_DRIVER=store-dynamodb
export IMPOSTER_DYNAMODB_TABLE=imposter-store
export AWS_ENDPOINT_URL=http://localhost:8000
go run cmd/imposter/main.go -configDir examples/rest/rate-limiting
```

## Advanced Testing Scenarios

### 1. Distributed Instance Simulation
Test behavior across multiple server instances:

```bash
# Terminal 1: Start first instance on port 8080
go run cmd/imposter/main.go -configDir examples/rest/rate-limiting -port 8080

# Terminal 2: Start second instance on port 8081  
go run cmd/imposter/main.go -configDir examples/rest/rate-limiting -port 8081

# Terminal 3: Load test both instances
hey -n 50 -c 10 http://localhost:8080/api/heavy &
hey -n 50 -c 10 http://localhost:8081/api/heavy &
wait
```

Note: Rate limiting is per-instance with in-memory store, but shared with Redis/DynamoDB stores.

### 2. TTL Cleanup Testing
Test automatic cleanup of expired rate limit entries:

```bash
# Set short TTL for testing
export IMPOSTER_RATE_LIMITER_TTL=10

# Generate load, then wait for cleanup
hey -n 20 -c 10 http://localhost:8080/api/heavy
sleep 15
hey -n 20 -c 10 http://localhost:8080/api/heavy
```

### 3. Failure Recovery Testing
Test behavior when store backend fails:

```bash
# Start with Redis
export IMPOSTER_STORE_DRIVER=store-redis
go run cmd/imposter/main.go -configDir examples/rest/rate-limiting &

# Stop Redis mid-test to see graceful degradation
docker stop <redis-container-id>

# Requests should continue working (rate limiting disabled on store failure)
hey -n 50 -c 10 http://localhost:8080/api/heavy
```

## Monitoring and Observability

Watch the logs for rate limiting events:

```bash
# Run with debug logging
export LOG_LEVEL=DEBUG
go run cmd/imposter/main.go -configDir examples/rest/rate-limiting
```

Look for log messages like:
- `rate limit exceeded for resource GET:/api/heavy: 4 > 3`
- `rate limit applied for resource GET:/api/heavy`
- `cleaned up expired instance data: instance:GET:/api/heavy:server-123`

## Features Demonstrated

1. **Progressive Rate Limiting**: Multiple tiers with different responses
2. **Response Variety**: Delays, error codes, custom headers, JSON responses  
3. **Store Backend Flexibility**: Works with in-memory, Redis, and DynamoDB
4. **Distributed Architecture**: Handles multiple server instances
5. **Automatic Cleanup**: TTL-based cleanup of stale entries
6. **Graceful Degradation**: Continues working if store backend fails
7. **Load Testing Integration**: Easy testing with standard tools like `hey`

This example provides a realistic simulation of API rate limiting that you can adapt for your own use cases.
