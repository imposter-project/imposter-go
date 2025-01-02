# Imposter-Go [![CI](https://github.com/imposter-project/imposter-go/actions/workflows/ci.yml/badge.svg)](https://github.com/imposter-project/imposter-go/actions/workflows/ci.yml)

A Go implementation of the [Imposter Mock Engine](https://www.imposter.sh).

## Features

- Local: Lightweight local HTTP mock server
- AWS Lambda: low latency, high throughput, ~15ms cold start
- [REST/HTTP API mock](https://docs.imposter.sh/rest_plugin/) support
- [SOAP/WSDL mock](https://docs.imposter.sh/soap_plugin/) support

## ⚠️ Limitations

- No support for [scripting](https://docs.imposter.sh/scripting/)
- No support (yet) for [OpenAPI](https://docs.imposter.sh/openapi_plugin/) mocks

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
   go run ./cmd/imposter/main.go ./examples/rest/simple
   ```

3. Visit `http://localhost:8080/hello` in your browser or use `curl`:
   ```bash
   curl http://localhost:8080/hello
   ```

## Configuration

Set the `IMPOSTER_PORT` environment variable to change the default port:
```bash
export IMPOSTER_PORT=9090
```

## Examples

The repository includes several examples demonstrating different features:

- [Simple REST API](examples/simple) - Basic REST API mocking
- [SOAP Web Service](examples/soap/simple) - SOAP/WSDL-based service mocking
- [AWS Lambda](examples/lambda) - Running Imposter in AWS Lambda

## Testing

Run tests using:
```bash
go test ./...
```