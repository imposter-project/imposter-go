# Concurrent Request Rate Limiter Design

## Overview

The rate limiter in imposter-go provides concurrent request limiting functionality that tracks active requests per resource and applies different responses based on configurable thresholds. It operates as a global singleton that integrates seamlessly with both REST and SOAP plugins, using atomic counters for thread-safe operation across multiple server instances.

## Architecture

### Core Components

#### 1. RateLimiter Interface
```go
type RateLimiter interface {
    CheckAndIncrement(resourceKey string, limits []config.ConcurrencyLimit) (*config.ConcurrencyLimit, error)
    Decrement(resourceKey string) error
}
```

#### 2. Global Rate Limiter Instance
The rate limiter operates as a global singleton accessible via `GetGlobalRateLimiter()`, ensuring consistent state management across all requests and plugins.

#### 3. Store Integration
Rate limiting data is stored using the configured store provider:
- **InMemory**: Thread-safe atomic operations with mutex protection
- **Redis**: Native atomic increment/decrement operations with TTL
- **DynamoDB**: Atomic UpdateItem operations with TTL attribute support

### Request Processing Flow

```
Request → Plugin Handler → Rate Limit Check → Counter Increment → Limit Evaluation
                                    ↓
                          Rate Limited? → Process Limit Response → Return
                                    ↓ No
                          Normal Processing → Response Writing → Counter Decrement
```

## Implementation Details

### 1. Resource Key Generation

Resources are identified using a hash-based unique key format that considers all matching criteria:

```go
func GenerateResourceKey(method, name string, matcher *config.RequestMatcher) string {
    if method == "" {
        method = "*"
    }
    if name == "" {
        name = "*"
    }
    baseKey := fmt.Sprintf("%s:%s", strings.ToUpper(method), name)

    // If no additional matching criteria, use the simple key
    if matcher == nil || isEmptyMatcher(matcher) {
        return baseKey
    }

    // Generate hash of all matching criteria
    hash := generateMatcherHash(matcher)
    return fmt.Sprintf("%s:%s", baseKey, hash)
}
```

**Examples:**
- Simple: `GET:/api/users`, `POST:/api/orders` 
- With headers: `GET:/api/users:8afa6046`, `GET:/api/users:595d8f88`
- SOAP: `POST:getUserDetails`, `POST:getUserDetails:b59e2e91`

#### Hash Generation
The hash includes all `RequestMatcher` criteria in deterministic order:
- RequestHeaders, QueryParams, FormParams, PathParams
- RequestBody matching conditions  
- SOAP-specific fields (SOAPAction, Binding)
- Expression conditions (AllOf, AnyOf)

This ensures that resources with identical method/path but different matching criteria receive separate rate limit counters, solving the multi-config collision issue.

### 2. Atomic Counter Management

The rate limiter uses a single atomic counter per resource stored with the key pattern:
```
counter:{resourceKey}
```

**Counter Operations:**
- **Increment**: Atomic increment when request starts
- **Rollback**: Atomic decrement if rate limit exceeded  
- **Cleanup**: Atomic decrement when request completes

### 3. Concurrency Limit Evaluation

Limits are evaluated using "greater than" logic with highest matching threshold selection:

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

**Example:** With limits `[3, 5, 10]` and current count `7`:
- Count `7 > 3` ✓ (match)
- Count `7 > 5` ✓ (match, overwrites previous)
- Count `7 > 10` ✗ (no match)
- Returns limit with threshold `5`

### 4. Plugin Integration

#### REST Plugin Integration
```go
// Check rate limiting if configured
if len(best.Resource.Concurrency) > 0 {
    processResponseFunc := func(exch *exchange.Exchange, requestMatcher *config.RequestMatcher, response *config.Response, respProc response.Processor) {
        h.processResponse(exch, requestMatcher, response, respProc)
    }

    shouldLimit := common.RateLimitCheck(
        best.Resource,
        best.Resource.Method,    // HTTP method (GET, POST, etc.)
        best.Resource.Path,      // Resource path (/api/users)
        exch,
        respProc,
        processResponseFunc,
    )

    if shouldLimit {
        return
    }
}
```

#### SOAP Plugin Integration
```go
// Check rate limiting if configured
if len(best.Resource.Concurrency) > 0 {
    processResponseFunc := func(exch *exchange.Exchange, requestMatcher *config.RequestMatcher, response *config.Response, respProc response.Processor) {
        h.processResponse(exch, bodyHolder, requestMatcher, response, op, respProc)
    }

    shouldLimit := common.RateLimitCheck(
        best.Resource,
        "POST",                  // SOAP always uses POST
        op.Name,                 // SOAP operation name
        exch,
        respProc,
        processResponseFunc,
    )

    if shouldLimit {
        return
    }
}
```

### 5. Cleanup Mechanism

The rate limiter uses a deferred cleanup approach to ensure counters are decremented after response processing:

```go
// Register cleanup function with ResponseState
cleanupFunc := func() {
    if err := rateLimiter.Decrement(resourceKey); err != nil {
        logger.Warnf("failed to decrement rate limiter count: %v", err)
    }
}
exch.ResponseState.CleanupFunctions = append(exch.ResponseState.CleanupFunctions, cleanupFunc)
```

**Cleanup Execution Flow:**
1. Request processed normally
2. Response written to client via `WriteToResponseWriter()`
3. Cleanup functions executed automatically
4. Counter decremented atomically

### 6. Store Implementation Details

#### Key Patterns
- **Counter Key**: `counter:{resourceKey}` (e.g., `counter:GET:/api/users`)
- **Global Prefix**: Applied automatically via store provider
- **TTL Support**: Automatic cleanup via store-specific TTL mechanisms

#### Store Provider Capabilities

**InMemory Store:**
- Mutex-protected atomic operations
- TTL support via `IMPOSTER_STORE_INMEMORY_TTL`
- Thread-safe for concurrent access

**Redis Store:**
- Native `INCRBY`/`DECRBY` atomic operations
- TTL via `EXPIRE` command
- High-performance distributed counter

**DynamoDB Store:**
- `UpdateItem` with `ADD` operation for atomic increments
- TTL attribute for automatic cleanup
- Consistent atomic operations across regions

## Configuration

### Resource Configuration

#### Basic Rate Limiting
```yaml
plugin: rest
resources:
- path: /api/users
  method: GET
  concurrency:
    - limit: 5
      response:
        statusCode: 429
        content: "Too many concurrent requests"
  response:
    statusCode: 200
    content: "Success"
```

#### Multiple Thresholds
```yaml
plugin: rest
resources:
- path: /api/heavy-operation
  method: POST
  concurrency:
    - limit: 2
      response:
        delay:
          exact: 1000
    - limit: 5
      response:
        statusCode: 503
        headers:
          Retry-After: "30"
        content: "Service temporarily overloaded"
    - limit: 10
      response:
        statusCode: 429
        content: "Rate limit exceeded"
  response:
    statusCode: 202
    content: "Operation queued"
```

#### SOAP Rate Limiting
```yaml
plugin: soap
resources:
- operation: getUserDetails
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

### Environment Variables

| Variable | Purpose | Default | Example |
|----------|---------|---------|---------|
| `IMPOSTER_RATE_LIMITER_TTL` | Rate limiter data TTL (seconds) | `300` (5 minutes) | `IMPOSTER_RATE_LIMITER_TTL=600` |
| `IMPOSTER_STORE_DRIVER` | Store backend selection | `inmemory` | `IMPOSTER_STORE_DRIVER=store-redis` |
| `IMPOSTER_STORE_INMEMORY_TTL` | InMemory store TTL (seconds) | No TTL | `IMPOSTER_STORE_INMEMORY_TTL=300` |

## Error Handling and Resilience

### Fail-Open Design
The rate limiter follows a fail-open approach:
- Store failures allow requests to proceed
- Increment/decrement errors are logged but don't block requests
- Network issues with Redis/DynamoDB don't impact availability

### Rollback Protection
When rate limits are exceeded:
1. Counter is atomically incremented
2. Limit evaluation performed
3. If exceeded, counter is immediately rolled back
4. Ensures accurate counting even under high concurrency

### Cleanup Guarantees
- Cleanup functions always execute after response writing
- Failed decrements are logged but don't impact request processing
- Store-level TTL provides secondary cleanup mechanism

## Performance Characteristics

### Throughput Optimisation
- Single atomic operation per request (increment)
- Minimal store queries for limit evaluation
- Efficient key structures for store access patterns

### Memory Management
- TTL-based cleanup prevents unbounded growth
- Atomic counters are more efficient than per-instance tracking
- Store provider abstractions allow optimal backend selection

### Concurrency Safety
- All operations use atomic store operations
- No in-memory locks for distributed stores
- Thread-safe design for high-concurrency scenarios

## Monitoring and Observability

### Logging
- Rate limit applications logged at INFO level
- Store failures logged at WARN level
- Counter decrement failures logged at WARN level

## Testing Strategy

### Unit Testing
- Mock store providers for isolated rate limiter logic testing
- Limit matching algorithm verification
- Counter rollback scenarios
- Cleanup function execution

### Integration Testing
- All store backends (InMemory, Redis, DynamoDB)
- Concurrent request scenarios
- Multi-threshold limit configurations
- TTL behavior verification

### Performance Testing
- High-concurrency load testing
- Store backend performance comparison
- Memory usage under sustained load
- Cleanup efficiency measurement

## Key Implementation Files

- **`internal/ratelimiter/ratelimiter.go`**: Core rate limiter implementation
- **`internal/common/ratelimit.go`**: Common rate limiting function for plugin integration
- **`internal/system/instance.go`**: Instance ID generation utilities
- **`plugin/rest/handler.go`**: REST plugin rate limiting integration
- **`plugin/soap/handler.go`**: SOAP plugin rate limiting integration
- **`internal/store/`**: Store provider implementations with atomic operations

## Future Enhancement Opportunities

1. **Client-based Limiting**: Rate limiting per IP address, user ID, or custom headers
2. **Time-window Limiting**: Sliding window rate limiting in addition to concurrent request limiting  
3. **Dynamic Configuration**: Hot-reload of rate limiting rules without service restart
4. **Circuit Breaker Integration**: Automatic request blocking when backends are overloaded
5. **Advanced Metrics**: Detailed rate limiting metrics via system endpoints
6. **Pattern Matching**: Wildcard or regex-based resource matching for flexible configuration
