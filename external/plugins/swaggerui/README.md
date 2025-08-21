# SwaggerUI Plugin

A web-based interactive documentation viewer for OpenAPI and Swagger specifications as an external plugin for Imposter. This plugin automatically serves OpenAPI specifications through an embedded Swagger UI interface, making it easy to explore and test your mock APIs.

## Features

- **Interactive API Documentation** with full Swagger UI functionality
- **OpenAPI 3.x and Swagger 2.0 Support** - handles both JSON and YAML formats
- **Automatic Server URL Injection** - automatically updates spec files with the current server URL
- **Multi-Spec Support** - displays multiple OpenAPI specifications in a single interface
- **Embedded Web Interface** - serves static assets from embedded filesystem
- **Caching and Performance** - caches processed specifications for optimal performance
- **Zero Configuration** - automatically discovers and serves all OpenAPI specs in your configuration

## File Structure

```
external/plugins/swaggerui/
├── plugin.go              # Core plugin implementation and routing
├── spec.go                # OpenAPI specification processing and serving
├── ui.go                  # Web UI serving and template processing
├── spec_test.go           # Tests for spec processing functionality
├── ui_test.go             # Tests for UI functionality
└── www/                   # Embedded web assets
    ├── index.html.tmpl     # HTML template for the main UI
    ├── swagger-ui.css      # Swagger UI styles
    ├── swagger-ui*.js      # Swagger UI JavaScript bundles
    ├── favicon-*.png       # Favicon files
    └── oauth2-redirect.html # OAuth2 redirect handler
```

## Configuration

The SwaggerUI plugin works alongside OpenAPI plugin configurations to provide interactive documentation. You need both an OpenAPI plugin configuration (with a spec file) and a SwaggerUI plugin configuration.

### Basic Usage

As shown in the `examples/swaggerui/` directory, you need two configuration files:

**OpenAPI Configuration (`imposter-config.yaml`):**
```yaml
plugin: openapi
specFile: petstore30.yaml
```

**SwaggerUI Configuration (`swaggerui-config.yaml`):**
```yaml
plugin: swaggerui
```

The SwaggerUI plugin will automatically:
1. Detect the `petstore30.yaml` specification from the OpenAPI plugin
2. Serve it at `/_spec/openapi/petstore30.yaml`
3. Create an interactive web interface at `/_spec/`

### Multiple Specifications

```yaml
# petstore-config.yaml
plugin: openapi
specFile: specs/petstore.yaml
resources: []
---
# users-config.yaml  
plugin: openapi
specFile: specs/users.yaml
resources: []
---
# swaggerui-config.yaml
plugin: swaggerui
```

All OpenAPI specifications will be available in the SwaggerUI interface with automatic discovery.

## Usage

### 1. Enable External Plugins

```bash
export IMPOSTER_EXTERNAL_PLUGINS=true
```

### 2. Create Configuration Files

Create both OpenAPI and SwaggerUI plugin configurations (see `examples/swaggerui/` for a complete example):

**OpenAPI Configuration:**
```yaml
plugin: openapi
specFile: api-spec.yaml
resources: []
```

**SwaggerUI Configuration:**
```yaml
plugin: swaggerui
```

### 3. Run Imposter

```bash
make run /path/to/your/config
```

### 4. Access SwaggerUI Interface

Navigate to `http://localhost:8080/_spec/` to view the interactive documentation.

## API Endpoints

### SwaggerUI Interface

**Endpoint:** `GET /_spec/`

Opens the interactive Swagger UI interface showing all discovered OpenAPI specifications.

```bash
# Open in browser
open http://localhost:8080/_spec/
```

### Raw OpenAPI Specifications

**Endpoint:** `GET /_spec/openapi/{spec-filename}`

Serves the raw OpenAPI specification with server URLs automatically injected.

```bash
# Get processed OpenAPI spec
curl http://localhost:8080/_spec/openapi/petstore.yaml
```

**Example Response (OpenAPI 3.x):**
```json
{
  "openapi": "3.0.0",
  "info": {
    "title": "Petstore API",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "http://localhost:8080"
    }
  ],
  "paths": {
    "/pets": {
      "get": {
        "summary": "List pets",
        "responses": {
          "200": {
            "description": "List of pets"
          }
        }
      }
    }
  }
}
```

### Static Assets

**Endpoint:** `GET /_spec/{asset-path}`

Serves static Swagger UI assets (CSS, JavaScript, images).

```bash
# Get Swagger UI CSS
curl http://localhost:8080/_spec/swagger-ui.css

# Get Swagger UI JavaScript bundle  
curl http://localhost:8080/_spec/swagger-ui-bundle.js
```

## Server URL Injection

The plugin automatically modifies OpenAPI specifications to include the current server URL:

### OpenAPI 3.x Format
For OpenAPI 3.x specifications, the server URL is prepended to the `servers` array:

```yaml
# Original spec
openapi: 3.0.0
info:
  title: My API
  version: 1.0.0
paths: {}

# Becomes (when served)
{
  "openapi": "3.0.0",
  "info": {
    "title": "My API", 
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "http://localhost:8080"
    }
  ],
  "paths": {}
}
```

### Swagger 2.0 Format
For Swagger 2.0 specifications, the server URL is split into `host`, `basePath`, and `schemes`:

```yaml
# Original spec
swagger: '2.0'
info:
  title: My API
  version: 1.0.0
paths: {}

# Becomes (when served at https://api.example.com/v1)
{
  "swagger": "2.0",
  "info": {
    "title": "My API",
    "version": "1.0.0" 
  },
  "host": "api.example.com",
  "basePath": "/v1",
  "schemes": ["https"],
  "paths": {}
}
```

## Environment Variables

### IMPOSTER_OPENAPI_SPEC_PATH_PREFIX

Customizes the URL prefix for the SwaggerUI interface.

```bash
export IMPOSTER_OPENAPI_SPEC_PATH_PREFIX="/docs"
# SwaggerUI will be available at /docs/ instead of /_spec/
```

**Default:** `/_spec`

## Error Handling

The plugin provides appropriate HTTP status codes for different scenarios:

### Common Responses

- **200 OK** - Successful content delivery
- **302 Found** - Redirect from `/_spec` to `/_spec/`
- **404 Not Found** - Requested file or spec doesn't exist
- **405 Method Not Allowed** - Non-GET requests to SwaggerUI endpoints
- **500 Internal Server Error** - File reading or processing errors

### Error Examples

```bash
# Non-existent specification
curl http://localhost:8080/_spec/openapi/missing.yaml
# Returns: 404 Not Found

# Invalid HTTP method
curl -X POST http://localhost:8080/_spec/
# Returns: 405 Method Not Allowed

# Malformed OpenAPI spec
# Returns: 500 Internal Server Error with parsing details
```

## Performance and Caching

The plugin implements several performance optimizations:

### Specification Caching
- Processed OpenAPI specs are cached in memory
- Cached until server restart
- Automatic server URL injection applied once

### Static Asset Embedding
- All Swagger UI assets are embedded in the plugin binary
- No filesystem access required for static content
- Optimal loading performance

### Concurrent Safety
- Thread-safe caching with read-write mutex
- Multiple simultaneous requests handled efficiently

## File Format Support

### Supported Formats

**OpenAPI Specifications:**
- OpenAPI 3.0.x (JSON/YAML)
- OpenAPI 3.1.x (JSON/YAML) 
- Swagger 2.0 (JSON/YAML)

**Content Detection:**
- Automatic YAML/JSON format detection
- File extension agnostic parsing
- Robust error handling for malformed specs

### Example Supported Files

```bash
# All these formats are supported
api-spec.yaml          # YAML OpenAPI 3.x
api-spec.yml           # YAML OpenAPI 3.x  
api-spec.json          # JSON OpenAPI 3.x
swagger.yaml           # YAML Swagger 2.0
swagger.json           # JSON Swagger 2.0
petstore-openapi.yaml  # Any valid OpenAPI file
```

## Integration Examples

### Complete OpenAPI Mock Setup

Following the pattern shown in `examples/swaggerui/`:

**OpenAPI Configuration (`imposter-config.yaml`):**
```yaml
plugin: openapi
specFile: petstore30.yaml
```

**SwaggerUI Configuration (`swaggerui-config.yaml`):**
```yaml
plugin: swaggerui
```

**OpenAPI Specification (`petstore30.yaml`):**
```yaml
openapi: 3.0.0
info:
  title: Petstore API
  version: 1.0.0
paths:
  /pets:
    get:
      summary: List all pets
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              example:
                - id: 1
                  name: "Fluffy"
                  type: "cat"
```

**Access Points:**
- Mock API: `http://localhost:8080/pets`
- SwaggerUI: `http://localhost:8080/_spec/`
- Raw Spec: `http://localhost:8080/_spec/openapi/petstore30.yaml`

### Multiple API Documentation

```yaml
# users-config.yaml
plugin: openapi
specFile: specs/users-api.yaml
resources: []
---
# products-config.yaml
plugin: openapi  
specFile: specs/products-api.yaml
resources: []
---
# orders-config.yaml
plugin: openapi
specFile: specs/orders-api.yaml  
resources: []
---
# swaggerui-config.yaml
plugin: swaggerui
```

All three APIs will appear in the SwaggerUI interface with a dropdown selector.

## Building

The plugin is built as part of the main Imposter build:

```bash
make build-plugins
```

This creates `bin/plugin-swaggerui` which is automatically discovered by Imposter when external plugins are enabled.

## Implementation Notes

- **Automatic Discovery**: Scans all loaded configurations for `specFile` references
- **Server URL Integration**: Uses Imposter's configured server URL for spec modification  
- **Template Processing**: Uses Go's `text/template` for dynamic HTML generation
- **Embedded Assets**: All web assets bundled using Go's `embed` directive
- **Zero Dependencies**: Self-contained plugin with embedded Swagger UI distribution
