# Security Configuration Example

This example demonstrates how to configure security rules for your mock API endpoints using Imposter. The security configuration allows you to protect your endpoints with various authentication and authorisation checks.

## Configuration Features

The security configuration supports:

- Multiple security conditions with different matchers
- Request header checks (e.g., API keys, Bearer tokens)
- Query parameter validation
- Form parameter validation
- Default permit/deny behaviour
- Resource-level and global security rules

## Example Configuration

The `config.yaml` file in this directory shows how to:

1. Set up global security rules that apply to all endpoints
2. Configure resource-specific security rules
3. Use different types of matchers (exact match, regex, exists, etc.)
4. Combine multiple conditions
5. Handle authentication failures

## Security Configuration Structure

```yaml
# Global security rules (applied to all endpoints)
security:
  default: Deny  # Default effect if no conditions match (Deny or Permit)
  conditions:
    - effect: Permit  # Effect if condition matches (Permit or Deny)
      requestHeaders:
        Authorization: "Bearer token123"  # Exact match
        X-API-Key:  # Advanced matcher
          matcher:
            operator: Matches
            value: "^key-[0-9a-f]{8}$"
      queryParams:
        version: "v1"  # Exact match
      formParams:
        token: "valid-token"  # Exact match

# Resource-specific security rules
resources:
  - path: /protected
    security:  # Overrides global security for this resource
      default: Deny
      conditions:
        - effect: Permit
          requestHeaders:
            X-Resource-Key: "secret"
```

## Supported Matchers

You can use various matchers for your conditions:

1. Simple string match (exact match):
   ```yaml
   Authorization: "Bearer token123"
   ```

2. Advanced matchers:
   ```yaml
   X-API-Key:
     matcher:
       operator: Matches
       value: "^key-[0-9a-f]{8}$"
   ```

Available operators:
- `EqualTo` - Exact string match
- `NotEqualTo` - Inverse of exact match
- `Contains` - String contains value
- `NotContains` - String does not contain value
- `Matches` - Regular expression match
- `NotMatches` - Regular expression does not match
- `Exists` - Header/param exists (any value)
- `NotExists` - Header/param does not exist

## Running the Example

1. Start Imposter with this configuration:
   ```bash
   imposter -d .
   ```

2. Try accessing the protected endpoints:
   ```bash
   # Should return 401 Unauthorised
   curl http://localhost:8080/api

   # Should succeed with valid token
   curl -H "Authorization: Bearer token123" http://localhost:8080/api

   # Should succeed with valid API key
   curl -H "X-API-Key: key-12345678" http://localhost:8080/api

   # Resource-specific protection
   curl -H "X-Resource-Key: secret" http://localhost:8080/protected
   ```

## Security Best Practices

1. Always set a default effect (usually "Deny")
2. Use strong token formats and validation patterns
3. Consider using regex patterns for flexible matching
4. Combine multiple conditions for stronger security
5. Use resource-specific rules for granular control 