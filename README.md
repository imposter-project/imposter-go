# imposter-go [![CI](https://github.com/imposter-project/imposter-go/actions/workflows/ci.yml/badge.svg)](https://github.com/imposter-project/imposter-go/actions/workflows/ci.yml)

A Go implementation of the [Imposter Mock Engine](https://www.imposter.sh).

## Features

- 💻 Run locally: Lightweight local HTTP mock server
- 🚀 Run in AWS Lambda: low latency, high throughput, <140ms cold start (see [test results](./examples/lambda/perf-tests))
- ✅ [REST/HTTP API mock](https://docs.imposter.sh/rest_plugin/) support
- ✅ [SOAP/WSDL mock](https://docs.imposter.sh/soap_plugin/) support
- ✅ [OpenAPI/Swagger mock](https://docs.imposter.sh/openapi_plugin/) support
- ✅ JavaScript [scripting](https://docs.imposter.sh/scripting/)
- ✅ Support for [steps](https://docs.imposter.sh/steps/)

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
# For Intel Macs (x86_64)
curl -L https://github.com/imposter-project/imposter-go/releases/latest/download/imposter-go_Darwin_x86_64.tar.gz | tar xz
sudo mv imposter-go /usr/local/bin/

# For Apple Silicon Macs (arm64)
curl -L https://github.com/imposter-project/imposter-go/releases/latest/download/imposter-go_Darwin_arm64.tar.gz | tar xz
sudo mv imposter-go /usr/local/bin/
```

#### Linux

```bash
# For x86_64 systems
curl -L https://github.com/imposter-project/imposter-go/releases/latest/download/imposter-go_Linux_x86_64.tar.gz | tar xz
sudo mv imposter-go /usr/local/bin/

# For arm64 systems
curl -L https://github.com/imposter-project/imposter-go/releases/latest/download/imposter-go_Linux_arm64.tar.gz | tar xz
sudo mv imposter-go /usr/local/bin/
```

#### Windows

1. Download the latest release from [GitHub Releases](https://github.com/imposter-project/imposter-go/releases/latest)
2. Extract the `imposter-go_Windows_x86_64.zip` file
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
export IMPOSTER_PORT=9090
```

Enable recursive directory scanning for configuration files:
```bash
export IMPOSTER_CONFIG_SCAN_RECURSIVE=true
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

## Building from Source

### Requirements

- Go 1.23 or later (earlier versions may work but are not tested)
- Make (for building)

### Build Steps

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

## Testing

Run the test suite using:
```bash
make test
```

This will run all tests.

To run a specific test, use the `go test` command:
```bash
go test ./pkg/server
```

To run tests with verbose output, use the `-v` flag:
```bash
go test -v ./pkg/server
```

---

## Releasing

To create a new release:

1. Tag your commit with a semver tag prefixed with 'v':
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. The GitHub Actions workflow will automatically:
   - Build binaries for all supported platforms (Linux, macOS, Windows)
   - Generate a changelog from commit messages
   - Create a GitHub release
   - Upload the built artifacts

The release notes will include:
- Version and release date
- Automatically generated changelog (excluding docs, test, ci, and chore commits)
- Installation instructions
- Link to the full changelog comparing with the previous tag
