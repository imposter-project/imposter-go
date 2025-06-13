# Concurrent Request Rate Limiter Design

## Overview

This document outlines the design and implementation of a concurrent request rate limiter in imposter-go. The rate limiter tracks concurrent requests per resource and applies different responses based on configurable thresholds.

## Requirements

### Functional Requirements
1. **Concurrent Request Tracking**: Track the number of active requests per resource across multiple server instances
2. **Configurable Thresholds**: Support multiple concurrency limits per resource with different responses
3. **Distributed State**: Use configurable datastores (inmemory, DynamoDB, Redis) for state management
4. **TTL Cleanup**: Automatically clean up stale request counts from ungraceful shutdowns
5. **Per-Resource Configuration**: Configure rate limits individually per resource
6. **Threshold Matching**: Use "greater than" logic with highest matching limit selection
7. **Plugin Support**: Support both REST and SOAP plugins with different resource naming conventions

### Non-Functional Requirements
1. **High Performance**: Minimal latency impact on request processing
2. **Thread Safety**: Safe for concurrent request handling
3. **Fault Tolerance**: Handle store failures gracefully
4. **Scalability**: Support multiple server instances (Lambda, containers)

## Architecture

### High-Level Flow
```
Request → Rate Limiter Check → Plugin Processing → Response
           ↓
         Store Update
```

### Components

#### 1. Configuration Model
```go
type ConcurrencyLimit struct {
    Limit    int       `yaml:"limit" json:"limit"`
    Response *Response `yaml:"response" json:"response"`
}

type Resource struct {
    // ... existing fields
    Concurrency []ConcurrencyLimit `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
}
```

#### 2. Rate Limiter Core
```go
type RateLimiter interface {
    CheckAndIncrement(resourceKey string, limits []config.ConcurrencyLimit, instanceID string) (*config.ConcurrencyLimit, error)
    Decrement(resourceKey string, instanceID string) error
    Cleanup() error
}
```

#### 3. Common Rate Limiting Function
```go
func RateLimitCheck(
    resource *config.Resource,
    resourceMethod string,
    resourceName string,
    exch *exchange.Exchange,
    respProc response.Processor,
    processResponseFunc func(*exchange.Exchange, *config.RequestMatcher, *config.Response, response.Processor),
) (shouldLimit bool, cleanupFunc func())
```

#### 4. Store Keys Structure
```
Rate limiter uses the following key patterns in the store:
- `activity:{resourceKey}:{instanceID}` - Resource activity data per server instance
```

## Implementation Details

### 1. Request Processing Flow

The rate limiter is integrated into both REST and SOAP plugin handlers:

```go
// Check rate limiting if configured
if len(best.Resource.Concurrency) > 0 {
    processResponseFunc := func(exch *exchange.Exchange, requestMatcher *config.RequestMatcher, response *config.Response, respProc response.Processor) {
        h.processResponse(exch, requestMatcher, response, respProc)
    }

    shouldLimit, cleanupFunc := common.RateLimitCheck(
        best.Resource,
        resourceMethod,  // "POST" for SOAP, best.Resource.Method for REST
        resourceName,    // operation name for SOAP, path for REST
        exch,
        respProc,
        processResponseFunc,
    )

    if shouldLimit {
        return
    }

    if cleanupFunc != nil {
        defer cleanupFunc()
    }
}
```

### 2. Concurrency Checking Algorithm

```go
func (rl *RateLimiterImpl) CheckAndIncrement(resourceKey string, limits []config.ConcurrencyLimit, instanceID string) (*config.ConcurrencyLimit, error) {
    // Get current total active count
    totalActive, err := rl.getTotalActiveCount(resourceKey)
    if err != nil {
        // On error, allow the request to proceed
        return nil, nil
    }

    // Check if any limit is exceeded after incrementing
    futureTotal := totalActive + 1
    matchedLimit := rl.findMatchingLimit(futureTotal, limits)

    if matchedLimit != nil {
        // Rate limit exceeded, return the matched limit
        return matchedLimit, nil
    }

    // Increment the counter
    err = rl.incrementActiveCount(resourceKey, instanceID)
    if err != nil {
        // On error, allow the request to proceed
        return nil, nil
    }

    return nil, nil
}
```

### 3. Limit Matching Logic

The rate limiter uses "greater than" logic with highest matching limit selection:

```go
func (rl *RateLimiterImpl) findMatchingLimit(currentCount int, limits []config.ConcurrencyLimit) *config.ConcurrencyLimit {
    // Sort limits by threshold (ascending) for proper matching
    sortedLimits := make([]config.ConcurrencyLimit, len(limits))
    copy(sortedLimits, limits)
    sort.Slice(sortedLimits, func(i, j int) bool {
        return sortedLimits[i].Limit < sortedLimits[j].Limit
    })

    // Find the highest matching limit (> logic)
    var matchedLimit *config.ConcurrencyLimit
    for _, limit := range sortedLimits {
        if currentCount > limit.Limit {
            matchedLimit = &limit
        }
    }

    return matchedLimit
}
```

### 4. TTL and Cleanup Mechanism

**Resource Activity Tracking**:
```go
type ResourceActivity struct {
    Count     int       `json:"count"`
    Timestamp time.Time `json:"timestamp"`
}
```

**TTL Implementation**:
- InMemory: Periodic cleanup goroutine with configurable TTL (default 5 minutes)
- Redis: Native TTL support
- DynamoDB: TTL attribute support

### 5. Instance ID Generation

Instance IDs are generated using the system package for consistency:

```go
// internal/system/instance.go
func GenerateInstanceID() string {
    hostname, _ := os.Hostname()
    pid := os.Getpid()
    timestamp := time.Now().UnixNano()
    return fmt.Sprintf("%s-%d-%d", hostname, pid, timestamp)
}
```

### 6. Resource Key Generation

```go
func GenerateResourceKey(method, name string) string {
    if method == "" {
        method = "*"
    }
    if name == "" {
        name = "*"
    }
    return fmt.Sprintf("%s:%s", strings.ToUpper(method), name)
}
```

**Resource Naming by Plugin**:
- **REST**: Uses `resource.Path` (e.g., `/api/users`)
- **SOAP**: Uses operation name (e.g., `getPetById`)

## Plugin Integration

### REST Plugin Integration
```go
shouldLimit, cleanupFunc := common.RateLimitCheck(
    best.Resource,
    best.Resource.Method,
    best.Resource.Path, // resourceName (path for REST)
    exch,
    respProc,
    processResponseFunc,
)
```

### SOAP Plugin Integration
```go
shouldLimit, cleanupFunc := common.RateLimitCheck(
    best.Resource,
    "POST",  // defaultMethod for SOAP
    op.Name, // resourceName (operation name)
    exch,
    respProc,
    processResponseFunc,
)
```

## Store Implementation Details

### Resource Activity Data Structure
Each server instance maintains its own activity count with timestamp:
```json
{
    "count": 3,
    "timestamp": "2023-10-01T12:00:00Z"
}
```

### Store Key Patterns
- Activity Key: `activity:{resourceKey}:{instanceID}`
- Example: `activity:GET:/api/users:server1-1234-1696156800000000000`

### InMemory Store
- Uses periodic cleanup goroutine
- Configurable TTL via `IMPOSTER_RATE_LIMITER_TTL` environment variable
- Thread-safe with mutex protection

### Redis Store
- Uses native TTL for automatic cleanup
- Atomic increment/decrement operations
- Efficient key pattern matching for cleanup

### DynamoDB Store
- Uses TTL attribute for automatic cleanup
- Atomic counter operations
- Efficient query patterns for activity aggregation

## Configuration Examples

### Basic Configuration
```yaml
plugin: rest
resources:
- path: /api/users
  method: get
  concurrency:
    - limit: 5
      response:
        delay:
          exact: 500
    - limit: 10  
      response:
        statusCode: 429
        content: "Too many concurrent requests"
  response:
    statusCode: 200
    content: "Success"
```

### SOAP Configuration
```yaml
plugin: soap
resources:
- operation: getPetById
  concurrency:
    - limit: 3
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
```

### Advanced Configuration
```yaml
plugin: rest
resources:
- path: /api/heavy-operation
  method: post
  concurrency:
    - limit: 2
      response:
        delay:
          range:
            min: 1000
            max: 3000
    - limit: 5
      response:
        statusCode: 503
        headers:
          Retry-After: "30"
        content: "Service temporarily overloaded"
  response:
    statusCode: 202
    content: "Operation queued"
```

## Error Handling

1. **Store Failures**: Log errors but allow requests to proceed
2. **Invalid Configuration**: Validate during startup
3. **Cleanup Failures**: Log warnings but continue processing
4. **Race Conditions**: Use atomic operations and proper locking

## Performance Considerations

1. **Atomic Operations**: Use store-native atomic increment/decrement
2. **Efficient Aggregation**: Minimize store queries for total count calculation
3. **Cleanup Frequency**: Balance between accuracy and performance (default 1-minute cleanup interval)
4. **Store Selection**: Redis recommended for high-throughput scenarios
5. **Memory Usage**: TTL cleanup prevents unbounded memory growth

## Testing Strategy

### Unit Tests
- Rate limiter logic with mock stores
- Limit matching algorithms
- TTL and cleanup mechanisms
- Configuration validation

### Integration Tests
- All store backends (InMemory, Redis, DynamoDB)
- Multi-instance scenarios
- TTL behavior verification
- Cross-store compatibility

### Performance Tests
- Concurrent request handling
- Store backend performance comparison
- Memory usage under load
- Cleanup efficiency

## Migration and Deployment

1. **Backward Compatibility**: Feature is opt-in via configuration
2. **Gradual Rollout**: Can be enabled per resource
3. **Monitoring**: Logs rate limiting events for observability
4. **Configuration Validation**: Startup validation prevents misconfigurations

## Future Enhancements

1. **Client-based Limiting**: Rate limit per IP, user ID, or header value
2. **Sliding Window**: Time-based rate limiting in addition to concurrent requests
3. **Metrics Integration**: Expose rate limiting metrics via system endpoints
4. **Dynamic Configuration**: Hot-reload rate limiting rules without restart
5. **Circuit Breaker Integration**: Combine with circuit breaker patterns
6. **Advanced Matching**: Regular expressions or wildcard patterns for resource matching

## Appendix

### Environment Variables
- `IMPOSTER_RATE_LIMITER_TTL`: TTL in seconds for activity cleanup (default: 300)
- `IMPOSTER_STORE_DRIVER`: Store backend selection (inmemory, store-redis, store-dynamodb)

### Key Implementation Files
- `internal/ratelimiter/ratelimiter.go`: Core rate limiter implementation
- `internal/common/ratelimit.go`: Common rate limiting function for plugins
- `internal/system/instance.go`: Instance ID generation utilities
- `plugin/rest/handler.go`: REST plugin integration
- `plugin/soap/handler.go`: SOAP plugin integration