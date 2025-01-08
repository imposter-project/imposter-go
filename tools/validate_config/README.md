# Imposter Config Validator

A tool to validate Imposter configuration files against a JSON schema.

## Python
### Setup

1. Create and activate the virtual environment:
```bash
python3 -m venv venv
source venv/bin/activate  # On Unix/macOS
# or
.\venv\Scripts\activate  # On Windows
```

2. Install dependencies:
```bash
pip install -r requirements.txt
```

### Usage

Run the validator by providing a directory containing config files to validate:

```bash
./validate_configs.py <directory>
```

For example:
```bash
# Validate configs in the examples directory
./validate_configs.py ../../examples

# Validate configs in the all-examples directory
./validate_configs.py ../../all-examples

# Validate configs in both directories
./validate_configs.py ../../examples && ./validate_configs.py ../../all-examples
```

The script will find all files ending in `-config.yaml` or `-config.yml` in the specified directory (recursively) and validate them against the schema.

## GoLang

### Setup

Run
```bash
go build -o validate_configs
```

### Usage

Run the validator by providing a directory containing config files to validate:

```bash
./validate_configs -c <directory>



## Schema

The schema is defined in `imposter-config-schema.json` and supports:
- REST and SOAP configurations
- HBase configurations
- OpenAPI configurations
- Various matching conditions (EqualTo, NotEqualTo, etc.)
- String and numeric values
- Response templates
- Stores and XML namespaces

The schema uses JSON Schema inheritance to represent the type hierarchy:
- `matchCondition` - base type for all matchers (can be string/number or object)
- `bodyMatchCondition` - extends `matchCondition` with JSON/XML path support
- `requestBody` - extends `bodyMatchCondition` with allOf/anyOf support
- `requestMatcher` - base type for resources and interceptors


## Example Output

```
Validating 69 config files...
✓ /path/to/config1.yaml - Valid
✓ /path/to/config2.yaml - Valid
...

Summary: 69/69 files valid
```

If a file is invalid, the output will show the error and its location in the file:
```
✗ /path/to/config.yaml - Invalid:
  Error: 'value' is not of type 'string'
  Path: resources -> 0 -> requestBody -> value
``` 