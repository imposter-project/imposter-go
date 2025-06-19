# Development

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

See the [release process documentation](./release.md).
