# imposter-go [![CI](https://github.com/imposter-project/imposter-go/actions/workflows/ci.yml/badge.svg)](https://github.com/imposter-project/imposter-go/actions/workflows/ci.yml)

A Go implementation of the [Imposter Mock Engine](https://www.imposter.sh). This project is now considered stable.

## Features

- 💻 Run locally: Lightweight local HTTP mock server
- 🚀 Run in AWS Lambda: low latency, high throughput, <140ms cold start (see [test results](./examples/lambda/perf-tests))
- ✅ [REST/HTTP API mock](https://docs.imposter.sh/rest_plugin/) support
- ✅ [SOAP/WSDL mock](https://docs.imposter.sh/soap_plugin/) support
- ✅ [OpenAPI/Swagger mock](https://docs.imposter.sh/openapi_plugin/) support
- ✅ JavaScript [scripting](https://docs.imposter.sh/scripting/)
- ✅ Support for [steps](https://docs.imposter.sh/steps/)
- ✅ Support for [simulated delays](https://docs.imposter.sh/performance_simulation/), [simulated errors](https://docs.imposter.sh/failure_simulation/) and [rate limiting](./docs/rate_limiting.md)

## ⚠️ Limitations

- No support for Groovy [scripting](https://docs.imposter.sh/scripting/)
- No support (yet) for some SOAP styles (https://github.com/imposter-project/imposter-go/issues/7)

## Getting Started

The recommended way to get started for most users is to use the latest [Imposter CLI](https://github.com/imposter-project/imposter-cli).

> **Note**
> If you don't have Imposter CLI installed, or you want to use the `imposter-go` binary directly, continue to the [Installation](#Installation) section.

With Imposter CLI installed, use the `-t golang` option, and pass `-v <version>`:

```
imposter up -t golang -v <version>
```

...where `version` is from the [Releases](https://github.com/imposter-project/imposter-go/releases/) page, for example:

```
imposter up -t golang -v 0.15.0
```

## Installation

### Using Pre-built Binaries

Download the latest release for your platform from GitHub:

#### macOS

```bash
# For Intel Macs (amd64)
curl -L https://github.com/imposter-project/imposter-go/releases/latest/download/imposter-go_darwin_amd64.tar.gz | tar xz
sudo mv imposter-go /usr/local/bin/

# For Apple Silicon Macs (arm64)
curl -L https://github.com/imposter-project/imposter-go/releases/latest/download/imposter-go_darwin_arm64.tar.gz | tar xz
sudo mv imposter-go /usr/local/bin/
```

#### Linux

```bash
# For amd64 systems
curl -L https://github.com/imposter-project/imposter-go/releases/latest/download/imposter-go_linux_amd64.tar.gz | tar xz
sudo mv imposter-go /usr/local/bin/

# For arm64 systems
curl -L https://github.com/imposter-project/imposter-go/releases/latest/download/imposter-go_linux_arm64.tar.gz | tar xz
sudo mv imposter-go /usr/local/bin/
```

#### Windows

1. Download the latest release from [GitHub Releases](https://github.com/imposter-project/imposter-go/releases/latest)
2. Extract the `imposter-go_windows_amd64.zip` file
3. Add the extracted `imposter-go.exe` to your PATH or move it to a directory in your PATH

## Usage

Run with a directory containing Imposter configuration file(s):

```bash
imposter-go ./examples/rest/simple
```

Visit `http://localhost:8080/hello` in your browser or use `curl`:

```bash
curl http://localhost:8080/hello
```

#### Examples

The repository includes several examples demonstrating different features:

- [Simple REST API](examples/rest/simple) - Basic REST API mocking
- [SOAP Web Service](examples/soap/simple) - SOAP/WSDL-based service mocking
- [OpenAPI HTTP API](examples/openapi/v30) - OpenAPI-based service mocking
- [AWS Lambda](examples/lambda) - Running Imposter in AWS Lambda

---

## Configuration

A subset of the Imposter [environment variables](https://docs.imposter.sh/environment_variables/) are supported. For example:

Set the `IMPOSTER_PORT` environment variable to change the default port:
```bash
export IMPOSTER_PORT=9090  # Default: 8080
```

Enable recursive directory scanning for configuration files:
```bash
export IMPOSTER_CONFIG_SCAN_RECURSIVE=true  # Default: false
```

Set the `IMPOSTER_SERVER_URL` environment variable to override the URL reported by the server:
```bash
export IMPOSTER_SERVER_URL=http://example.com  # Default: http://localhost:8080
```

Set the `IMPOSTER_LOG_LEVEL` environment variable to control logging verbosity:
```bash
export IMPOSTER_LOG_LEVEL=DEBUG  # Available levels: TRACE, DEBUG, INFO, WARN, ERROR
```

The default log level is DEBUG. Available log levels:
- TRACE - Most verbose, logs all messages
- DEBUG - Detailed information for debugging
- INFO - General operational messages
- WARN - Warning messages for potentially harmful situations
- ERROR - Error messages for serious problems

### Legacy Configuration Support

Imposter Go supports legacy configuration formats, for backwards compatibility with older Imposter configurations.

Enable support for legacy configuration format:
```bash
export IMPOSTER_SUPPORT_LEGACY_CONFIG=true
```

When legacy configuration support is enabled, older configuration formats are automatically transformed. For example:

```yaml
# Legacy format (root-level fields)
plugin: rest
path: /hello
contentType: text/plain
response:
  staticData: Hello, World!
```

```yaml
# Legacy format (deprecated names for resource-level fields)
plugin: rest
resources:
  - path: /hello
    contentType: application/json
    response:
      staticFile: response.json # Deprecated, use file instead
      staticData: Hello, World! # Deprecated, use content instead
      scriptFile: transform.js  # Deprecated, use a script step instead
```

---

## Development

See the [Development Guide](docs/development.md) for instructions on building, testing, and contributing to the project.
