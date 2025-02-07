# Steps Examples

This directory contains examples demonstrating the Steps feature in Imposter. Steps allow you to perform actions when receiving a request, such as executing scripts or making HTTP calls to external services.

## Files

## Examples in config.yaml

Demonstrates basic step usage in resources

1. **Simple Script Step** (`/api/log-request`)
   - Demonstrates a simple JavaScript step that logs request details
   - No external dependencies
   - Good starting point for understanding steps

2. **Weather API Integration** (`/api/weather/{city}`)
   - Shows how to call an external API using a remote step
   - Captures specific fields from the response
   - Uses captured data in the response template
   - Note: Requires a WeatherAPI key

3. **Multi-step Data Processing** (`/api/process-data`)
   - Demonstrates using multiple steps in sequence
   - Shows data flow between steps using stores
   - Includes input validation and transformation
   - Makes an external API call with transformed data

## Testing the Examples

### Basic Script Example
```bash
curl http://localhost:8080/api/log-request
```

### Weather API Example
```bash
# Replace YOUR_API_KEY in config.yaml first
curl http://localhost:8080/api/weather/London
```

### Data Processing Example
```bash
curl -X POST http://localhost:8080/api/process-data \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "JOHN@EXAMPLE.COM"}'
```

## Notes

- Replace placeholder URLs (`http://external-service`, `http://auth-service`, etc.) with actual service URLs
- Replace `YOUR_API_KEY` with actual API keys where required
- The examples assume Imposter is running on `localhost:8080` 