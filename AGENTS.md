# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Conventions

- Use British English spelling for all function and variable names, comments and documentation.

## Development Commands

### Building and Running
- `make build` - Build the binary with proper version information
- `make run <config-path>` - Compile and run the server with a configuration directory (development mode)
- `make install` - Install the binary to your Go bin directory
- `make fmt` - Format all Go code
- `make build-plugins` - Build external plugins (currently swaggerui)

### Testing
- `make test` - Run all tests
- `make coverage` - Run tests with coverage report
- `make coverage-html` - Generate HTML coverage report in coverage.html
- `go test ./path/to/package` - Run tests for a specific package
- `go test -v ./path/to/package` - Run tests with verbose output

### Common Development Workflow
```bash
# Run the server with an example config during development
make run ./examples/rest/simple

# Test your changes
make test

# Format code before committing
make fmt
```

## Architecture Overview

Imposter-go is a mock server engine that supports REST, SOAP, OpenAPI, gRPC and websockets mocking with a plugin-based architecture.

### Core Components

**Adapter Layer** (`internal/adapter/`):
- Supports multiple runtime modes: HTTP server and AWS Lambda
- `httpserver/` - Standard HTTP server implementation
- `awslambda/` - AWS Lambda runtime adapter
- Mode detection happens automatically at startup

**Plugin System** (`plugin/`):
- Plugin interface defines `HandleRequest()` for processing HTTP requests
- Built-in plugins: `rest/`, `openapi/`, `soap/`
- External plugins supported via `external/` directory using Go plugins
- Each plugin handles specific mock types and configuration formats

**Configuration System** (`internal/config/`):
- YAML-based configuration with support for legacy formats
- `model.go` defines the configuration structure
- `legacy.go` handles backward compatibility transformations
- Supports recursive directory scanning for config files

**Request Processing Flow**:
1. Request enters via adapter (HTTP server or Lambda)
2. Config loader discovers and parses YAML configuration files
3. Plugin loader instantiates appropriate plugins based on config
4. Request routed to matching plugin via `HandleRequest()`
5. Plugin processes request using matchers, scripts, and response processors
6. Response returned through adapter

**Key Internal Packages**:
- `internal/handler/` - HTTP request routing and CORS handling
- `internal/matcher/` - Request matching logic (path, headers, body)
- `internal/response/` - Response processing and templating
- `internal/steps/` - Multi-step request processing with scripting support
- `internal/store/` - Data persistence (in-memory, Redis, DynamoDB)
- `internal/template/` - Response templating engine

### Configuration Format

Imposter uses YAML configuration files with this structure:
- `plugin` - Specifies the plugin type (rest, openapi, soap)
- `resources` - Array of mock resource definitions with paths, methods, responses
- Support for scripting (JavaScript), templating, conditional responses
- Legacy format support can be enabled via `IMPOSTER_SUPPORT_LEGACY_CONFIG=true`

### Testing Strategy

- Integration tests in `test/integration_test.go` create temporary configs and test full request flow
- Unit tests throughout codebase test individual components
- Test utilities in `test/testutils/` provide helper functions
- Examples in `examples/` serve as both documentation and integration test cases

