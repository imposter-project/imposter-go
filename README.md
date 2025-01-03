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
- Make (for building)

## Getting Started

1. Clone the repository:
   ```bash
   git clone https://github.com/imposter-project/imposter-go.git
   cd imposter-go
   ```

2. Run the server with an example configuration:
   ```bash
   make run ./examples/rest/simple
   ```

3. Visit `http://localhost:8080/hello` in your browser or use `curl`:
   ```bash
   curl http://localhost:8080/hello
   ```

4. Check the server version:
   ```bash
   curl http://localhost:8080/system/status
   ```

## Configuration

Set the `IMPOSTER_PORT` environment variable to change the default port:
```bash
export IMPOSTER_PORT=9090
```

## Development

The project uses Make for building and development. The following targets are available:

- `make run <path>` - Run the server directly (useful during development)
- `make build` - Build the project with version information
- `make install` - Install the binary to your Go bin directory
- `make test` - Run tests with verbose output

For development, use `make run` which will compile and run the server in one step:
```bash
# Run with a specific configuration
make run ./examples/rest/simple

# Run with multiple arguments
make run --debug ./examples/soap/simple
```

For production or installation, use `make build` or `make install`.

The version information is automatically derived from git tags. When building from source:
- Released versions will show the git tag (e.g. "v1.0.0")
- Development builds will show the git commit hash
- If no git information is available, it will show "dev"

## Examples

The repository includes several examples demonstrating different features:

- [Simple REST API](examples/simple) - Basic REST API mocking
- [SOAP Web Service](examples/soap/simple) - SOAP/WSDL-based service mocking
- [AWS Lambda](examples/lambda) - Running Imposter in AWS Lambda

## Testing

Run the test suite using:
```bash
make test
```

This will run all tests with verbose output, showing the progress of each test case.