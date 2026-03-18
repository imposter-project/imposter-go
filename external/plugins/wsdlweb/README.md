# WSDL Web Plugin

A web-based interactive viewer for WSDL/SOAP service definitions as an external plugin for Imposter. This plugin serves WSDL files through an embedded web interface, making it easy to explore SOAP services, operations, messages, and bindings in your mocks.

## Features

- **Interactive WSDL Viewer** with service, port, and operation browsing
- **WSDL 1.1 and WSDL 2.0 Support** - handles both versions
- **SOAP 1.1 and SOAP 1.2** - displays SOAPAction, bindings, and addresses
- **Multi-WSDL Support** - displays multiple WSDL files with a dropdown selector
- **Raw XML View** - toggle to see the original WSDL XML
- **Embedded Web Interface** - serves static assets from embedded filesystem
- **Caching** - caches loaded WSDL files for optimal performance
- **Zero Configuration** - automatically discovers WSDL files from SOAP plugin configs

## File Structure

```
external/plugins/wsdlweb/
├── plugin.go              # Core plugin implementation and routing
├── wsdl.go                # WSDL file loading, caching, and serving
├── ui.go                  # Web UI serving and template processing
├── wsdl_test.go           # Tests for WSDL processing functionality
├── ui_test.go             # Tests for UI functionality
└── www/                   # Embedded web assets
    ├── index.html         # HTML page for the viewer
    ├── wsdl-web.css       # Viewer styles
    ├── wsdl-web.js        # WSDL parsing and rendering logic
    └── wsdl-initializer.js.tmpl  # Template for dynamic config injection
```

## Configuration

The WSDL Web plugin works alongside SOAP plugin configurations. You need both a SOAP plugin configuration (with a WSDL file) and a WSDL Web plugin configuration.

### Basic Usage

As shown in the `examples/wsdlweb/` directory:

**Configuration (`imposter-config.yaml`):**
```yaml
plugin: soap
wsdlFile: petstore.wsdl
---
plugin: wsdlweb
```

The WSDL Web plugin will automatically:
1. Detect the `petstore.wsdl` from the SOAP plugin configuration
2. Serve it at `/_wsdl/wsdl/petstore.wsdl`
3. Create an interactive web interface at `/_wsdl/`

### Multiple WSDLs

#### petstore-config.yaml

```yaml
plugin: soap
wsdlFile: petstore.wsdl
resources: []
---
plugin: wsdlweb
```

#### orders-config.yaml

```yaml
plugin: soap
wsdlFile: orders.wsdl
resources: []
---
plugin: wsdlweb
```

All WSDL files will be available in the viewer interface with a dropdown selector.

## Usage

### 1. Enable External Plugins

```bash
export IMPOSTER_EXTERNAL_PLUGINS=true
```

### 2. Create Configuration Files

Create both SOAP and WSDL Web plugin configurations (see `examples/wsdlweb/`):

```yaml
plugin: soap
wsdlFile: service.wsdl
resources: []
---
plugin: wsdlweb
```

### 3. Run Imposter

```bash
make run /path/to/your/config
```

### 4. Access WSDL Web Interface

Navigate to `http://localhost:8080/_wsdl/` to view the interactive WSDL documentation.

## API Endpoints

### WSDL Web Interface

**Endpoint:** `GET /_wsdl/`

Opens the interactive WSDL viewer showing all discovered WSDL files.

### Raw WSDL Files

**Endpoint:** `GET /_wsdl/wsdl/{wsdl-filename}`

Serves the raw WSDL XML file.

```bash
curl http://localhost:8080/_wsdl/wsdl/petstore.wsdl
```

## Environment Variables

### IMPOSTER_WSDL_SPEC_PATH_PREFIX

Customizes the URL prefix for the WSDL Web interface.

```bash
export IMPOSTER_WSDL_SPEC_PATH_PREFIX="/soap-docs"
# WSDL Web will be available at /soap-docs/ instead of /_wsdl/
```

**Default:** `/_wsdl`

## Building

The plugin is built as part of the main Imposter build:

```bash
make build-plugins
```

This creates `bin/plugin-wsdlweb` which is automatically discovered by Imposter when external plugins are enabled.
