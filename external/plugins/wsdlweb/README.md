# WSDL Web Plugin

An external plugin for Imposter that embeds [WSDL Web](https://github.com/wsdl-web/wsdl-web) — an open-source interactive WSDL/SOAP viewer (similar to Swagger UI for OpenAPI). This plugin allows users to explore SOAP/WSDL specs in their mocks.

## Features

- **Embeds WSDL Web v0.9.1** — the full WSDL Web dist bundle is embedded as static assets
- **Multiple WSDL Support** — automatically discovers all WSDL files from SOAP plugin configs and passes them as `urls` to WSDL Web with a dropdown switcher
- **Zero Configuration** — automatically detects `wsdlFile` entries from SOAP plugin configs
- **Locked UI** — URL input and file browse controls are hidden since WSDLs are pre-configured
- **Raw WSDL Serving** — serves WSDL files at `/_wsdl/wsdl/{filename}` for WSDL Web to fetch
- **Caching** — caches loaded WSDL files for optimal performance

## File Structure

```
external/plugins/wsdlweb/
├── plugin.go              # Core plugin implementation and routing
├── wsdl.go                # WSDL file loading, caching, and serving
├── ui.go                  # Web UI serving and template processing
├── wsdl_test.go           # Tests for WSDL processing functionality
├── ui_test.go             # Tests for UI functionality
└── www/                   # Embedded web assets
    ├── index.html         # HTML page embedding WSDL Web
    ├── wsdl-web.css       # WSDL Web v0.9.1 stylesheet
    ├── wsdl-web.js         # WSDL Web v0.9.1 bundle (includes React)
    ├── icons.svg           # WSDL Web icons
    ├── favicon.ico         # Favicon
    └── wsdl-initializer.js.tmpl  # Template for WsdlWeb.init() config injection
```

## Configuration

The WSDL Web plugin works alongside SOAP plugin configurations. Add a `wsdlweb` plugin section to your config:

```yaml
plugin: soap
wsdlFile: petstore.wsdl
---
plugin: wsdlweb
```

### Multiple WSDLs

All WSDL files from all SOAP configs are automatically discovered. WSDL Web displays a dropdown switcher in the top bar to toggle between them:

```yaml
# petstore-config.yaml
plugin: soap
wsdlFile: petstore.wsdl
---
# orders-config.yaml
plugin: soap
wsdlFile: orders.wsdl
---
plugin: wsdlweb
```

This produces the equivalent of:

```javascript
WsdlWeb.init(document.getElementById('wsdl-web'), {
  urls: [
    { label: 'petstore.wsdl', url: '/_wsdl/wsdl/petstore.wsdl' },
    { label: 'orders.wsdl', url: '/_wsdl/wsdl/orders.wsdl' },
  ],
  showUrlInput: false,
  showExploreButton: false,
  showBrowseButton: false,
})
```

## Usage

### 1. Enable External Plugins

```bash
export IMPOSTER_EXTERNAL_PLUGINS=true
```

### 2. Run Imposter

```bash
make run /path/to/your/config
```

### 3. Access WSDL Web Interface

Navigate to `http://localhost:8080/_wsdl/` to view the interactive WSDL documentation.

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /_wsdl/` | WSDL Web interactive viewer |
| `GET /_wsdl/wsdl/{filename}` | Raw WSDL XML file |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `IMPOSTER_WSDL_SPEC_PATH_PREFIX` | `/_wsdl` | URL prefix for WSDL Web interface |

## Updating WSDL Web

To update the embedded WSDL Web version:

1. Download the latest standalone zip from [WSDL Web releases](https://github.com/wsdl-web/wsdl-web/releases)
2. Extract `wsdl-web.js`, `wsdl-web.css`, `icons.svg`, and `favicon.ico` into `www/`
3. Rebuild the plugin
