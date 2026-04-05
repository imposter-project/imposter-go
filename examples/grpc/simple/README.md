# gRPC Plugin Example

This example demonstrates the `grpc` external plugin that mocks gRPC services using `.proto` files and JSON response definitions.

The gRPC plugin uses the core request processing pipeline, so standard features like request matching, interceptors, capture, steps, scripting, and response templating all work with gRPC resources.

## Features

- **Proto file parsing** at startup (no `protoc` required)
- **JSON-based responses** converted to protobuf automatically
- **Native gRPC support** via HTTP/2 cleartext (h2c)
- **Full pipeline support** — matching, interceptors, capture, steps, scripting, templating

## Configuration

The `config` block specifies proto files. Responses use the standard `resources` block:

```yaml
plugin: grpc

config:
  protoFiles:
    - "petstore.proto"

resources:
  - path: "/store.PetStore/GetPet"
    response:
      file: "get-pet-response.json"
  - path: "/store.PetStore/ListPets"
    response:
      file: "list-pets-response.json"
```

### Configuration Reference

| Field | Description |
|-------|-------------|
| `config.protoFiles` | List of `.proto` files to parse (relative to config directory) |
| `resources[].path` | gRPC method path, e.g. `/store.PetStore/GetPet` |
| `resources[].response.file` | Path to a JSON file containing the response body |
| `resources[].response.content` | Inline JSON response body (alternative to `file`) |
| `resources[].response.template` | Enable response templating (default: `false`) |

## Usage

1. **Build the main binary and plugins**:
   ```bash
   make build && make build-plugins
   ```

2. **Enable external plugins and run**:
   ```bash
   export IMPOSTER_EXTERNAL_PLUGINS=true
   export IMPOSTER_PLUGIN_DIR=./bin
   make run ./examples/grpc/simple
   ```

3. **Test with grpcurl**:

   Get a pet:
   ```bash
   grpcurl -plaintext -proto examples/grpc/simple/petstore.proto \
     -d '{"id": 1}' \
     localhost:8080 store.PetStore/GetPet
   ```

   Expected response:
   ```json
   {
     "id": 1,
     "name": "Fido",
     "species": "Dog"
   }
   ```

   List all pets:
   ```bash
   grpcurl -plaintext -proto examples/grpc/simple/petstore.proto \
     localhost:8080 store.PetStore/ListPets
   ```

   Expected response:
   ```json
   {
     "pets": [
       {"id": 1, "name": "Fido", "species": "Dog"},
       {"id": 2, "name": "Whiskers", "species": "Cat"},
       {"id": 3, "name": "Bubbles", "species": "Fish"}
     ]
   }
   ```

## Response Format

Response files contain JSON that matches the structure of the protobuf response message. The plugin converts JSON to protobuf wire format automatically using the message descriptors from the `.proto` file.

Since resources use the standard pipeline, you can use templating:

```yaml
resources:
  - path: "/store.PetStore/GetPet"
    response:
      content: '{"id": 1, "name": "Fido", "species": "Dog"}'
      template: true
```
