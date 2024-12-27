# Imposter-Go

A Go implementation of the Imposter tool (https://github.com/outofcoffee/imposter), designed for HTTP mocking.

## Features

- Lightweight HTTP server
- Mock response capabilities
- Configurable via environment variables

## Requirements

- Go 1.21 or later

## Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/imposter-project/imposter-go.git
   cd imposter-go
   ```

2. Build and run the server:
   ```bash
   go run cmd/imposter/main.go
   ```

3. Visit `http://localhost:8080` in your browser or use `curl`:
   ```bash
   curl http://localhost:8080
   ```

## Configuration

Set the `IMPOSTER_PORT` environment variable to change the default port:
```bash
export IMPOSTER_PORT=9090
```

## Testing

Run tests using:
```bash
go test ./...
```