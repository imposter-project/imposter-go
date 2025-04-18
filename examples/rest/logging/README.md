# Resource Logging Feature

This example demonstrates how to use the resource logging feature in Imposter. This feature allows you to configure log messages for resources and interceptors that will be output when they handle a request.

## Configuration

The logging feature is configured by adding a `log` property to any resource or interceptor:

```yaml
resources:
  - path: /example
    method: GET
    log: "This is a log message"
    response:
      # Response configuration...
```

## Template Support

Log messages support the full template syntax, allowing you to include dynamic data:

```yaml
log: "User ${context.request.pathParams.id} accessed the API at ${datetime.now.iso8601_datetime}"
```

### Available Template Variables

You can use any of the standard template variables in your log messages:

- `${context.request.*}` - Access request data (path, method, headers, body, etc.)
- `${stores.*}` - Access data from stores
- `${datetime.now.*}` - Access current date/time in various formats
- `${random.*}` - Generate random values
- `${system.*}` - Access system configuration

## Examples in This Configuration

The configuration in this example includes:

1. A resource that logs path parameter access:
   ```
   User details retrieved for ID: ${context.request.pathParams.id}
   ```

2. A resource that logs with datetime and headers:
   ```
   Error endpoint accessed at ${datetime.now.iso8601_datetime} from ${context.request.headers.User-Agent:-unknown}
   ```
   Note the use of `:-unknown` as a default value if the User-Agent header is not present.

3. An interceptor that logs all GET requests:
   ```
   Request intercepted - path: ${context.request.path}
   ```

## Testing

Run Imposter with this configuration and make requests to the endpoints to see the log messages:

```
imposter -d .
```

Then make requests:
```
curl http://localhost:8080/user/42
curl http://localhost:8080/error
```

The log messages will appear in the Imposter console output.