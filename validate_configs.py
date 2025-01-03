#!/usr/bin/env python3

import json
import os
import sys
from pathlib import Path
import yaml
from jsonschema import validate, ValidationError, Draft7Validator

# Define the JSON schema based on the Go structs
schema = {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "type": "object",
    "required": ["plugin"],
    "properties": {
        "plugin": {
            "type": "string",
            "enum": ["rest", "soap"]
        },
        "basePath": {
            "type": "string"
        },
        "wsdlFile": {
            "type": "string"
        },
        "resources": {
            "type": "array",
            "items": {
                "type": "object",
                "required": ["response"],
                "properties": {
                    "method": {"type": "string"},
                    "path": {"type": "string"},
                    "operation": {"type": "string"},
                    "soapAction": {"type": "string"},
                    "binding": {"type": "string"},
                    "queryParams": {
                        "type": "object",
                        "additionalProperties": {
                            "oneOf": [
                                {"type": "string"},
                                {
                                    "type": "object",
                                    "properties": {
                                        "value": {"type": "string"},
                                        "operator": {
                                            "type": "string",
                                            "enum": ["EqualTo", "NotEqualTo", "Exists", "NotExists", "Contains", "NotContains", "Matches", "NotMatches", ""]
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    "headers": {
                        "type": "object",
                        "additionalProperties": {
                            "oneOf": [
                                {"type": "string"},
                                {
                                    "type": "object",
                                    "properties": {
                                        "value": {"type": "string"},
                                        "operator": {
                                            "type": "string",
                                            "enum": ["EqualTo", "NotEqualTo", "Exists", "NotExists", "Contains", "NotContains", "Matches", "NotMatches", ""]
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    "requestBody": {
                        "type": "object",
                        "properties": {
                            "value": {"type": "string"},
                            "operator": {
                                "type": "string",
                                "enum": ["EqualTo", "NotEqualTo", "Exists", "NotExists", "Contains", "NotContains", "Matches", "NotMatches", ""]
                            },
                            "jsonPath": {"type": "string"},
                            "xPath": {"type": "string"},
                            "xmlNamespaces": {
                                "type": "object",
                                "additionalProperties": {"type": "string"}
                            },
                            "allOf": {
                                "type": "array",
                                "items": {
                                    "type": "object",
                                    "properties": {
                                        "value": {"type": "string"},
                                        "operator": {
                                            "type": "string",
                                            "enum": ["EqualTo", "NotEqualTo", "Exists", "NotExists", "Contains", "NotContains", "Matches", "NotMatches", ""]
                                        },
                                        "jsonPath": {"type": "string"},
                                        "xPath": {"type": "string"},
                                        "xmlNamespaces": {
                                            "type": "object",
                                            "additionalProperties": {"type": "string"}
                                        }
                                    }
                                }
                            },
                            "anyOf": {
                                "type": "array",
                                "items": {
                                    "type": "object",
                                    "properties": {
                                        "value": {"type": "string"},
                                        "operator": {
                                            "type": "string",
                                            "enum": ["EqualTo", "NotEqualTo", "Exists", "NotExists", "Contains", "NotContains", "Matches", "NotMatches", ""]
                                        },
                                        "jsonPath": {"type": "string"},
                                        "xPath": {"type": "string"},
                                        "xmlNamespaces": {
                                            "type": "object",
                                            "additionalProperties": {"type": "string"}
                                        }
                                    }
                                }
                            }
                        }
                    },
                    "formParams": {
                        "type": "object",
                        "additionalProperties": {
                            "oneOf": [
                                {"type": "string"},
                                {
                                    "type": "object",
                                    "properties": {
                                        "value": {"type": "string"},
                                        "operator": {
                                            "type": "string",
                                            "enum": ["EqualTo", "NotEqualTo", "Exists", "NotExists", "Contains", "NotContains", "Matches", "NotMatches", ""]
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    "pathParams": {
                        "type": "object",
                        "additionalProperties": {
                            "oneOf": [
                                {"type": "string"},
                                {
                                    "type": "object",
                                    "properties": {
                                        "value": {"type": "string"},
                                        "operator": {
                                            "type": "string",
                                            "enum": ["EqualTo", "NotEqualTo", "Exists", "NotExists", "Contains", "NotContains", "Matches", "NotMatches", ""]
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    "capture": {
                        "type": "object",
                        "additionalProperties": {
                            "type": "object",
                            "properties": {
                                "enabled": {"type": "boolean"},
                                "store": {"type": "string"},
                                "key": {
                                    "type": "object",
                                    "properties": {
                                        "pathParam": {"type": "string"},
                                        "queryParam": {"type": "string"},
                                        "formParam": {"type": "string"},
                                        "requestHeader": {"type": "string"},
                                        "expression": {"type": "string"},
                                        "const": {"type": "string"},
                                        "requestBody": {
                                            "type": "object",
                                            "properties": {
                                                "jsonPath": {"type": "string"},
                                                "xPath": {"type": "string"},
                                                "xmlNamespaces": {
                                                    "type": "object",
                                                    "additionalProperties": {"type": "string"}
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    },
                    "response": {
                        "type": "object",
                        "properties": {
                            "content": {"type": "string"},
                            "statusCode": {"type": "integer"},
                            "file": {"type": "string"},
                            "fail": {"type": "string"},
                            "delay": {
                                "type": "object",
                                "properties": {
                                    "exact": {"type": "integer"},
                                    "min": {"type": "integer"},
                                    "max": {"type": "integer"}
                                }
                            },
                            "headers": {
                                "type": "object",
                                "additionalProperties": {"type": "string"}
                            },
                            "template": {"type": "boolean"}
                        }
                    }
                }
            }
        },
        "interceptors": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "method": {"type": "string"},
                    "path": {"type": "string"},
                    "operation": {"type": "string"},
                    "soapAction": {"type": "string"},
                    "binding": {"type": "string"},
                    "queryParams": {
                        "type": "object",
                        "additionalProperties": {
                            "oneOf": [
                                {"type": "string"},
                                {
                                    "type": "object",
                                    "properties": {
                                        "value": {"type": "string"},
                                        "operator": {
                                            "type": "string",
                                            "enum": ["EqualTo", "NotEqualTo", "Exists", "NotExists", "Contains", "NotContains", "Matches", "NotMatches", ""]
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    "headers": {
                        "type": "object",
                        "additionalProperties": {
                            "oneOf": [
                                {"type": "string"},
                                {
                                    "type": "object",
                                    "properties": {
                                        "value": {"type": "string"},
                                        "operator": {
                                            "type": "string",
                                            "enum": ["EqualTo", "NotEqualTo", "Exists", "NotExists", "Contains", "NotContains", "Matches", "NotMatches", ""]
                                        }
                                    }
                                }
                            ]
                        }
                    },
                    "requestBody": {
                        "type": "object",
                        "properties": {
                            "value": {"type": "string"},
                            "operator": {
                                "type": "string",
                                "enum": ["EqualTo", "NotEqualTo", "Exists", "NotExists", "Contains", "NotContains", "Matches", "NotMatches", ""]
                            },
                            "jsonPath": {"type": "string"},
                            "xPath": {"type": "string"},
                            "xmlNamespaces": {
                                "type": "object",
                                "additionalProperties": {"type": "string"}
                            }
                        }
                    },
                    "response": {
                        "type": "object",
                        "properties": {
                            "content": {"type": "string"},
                            "statusCode": {"type": "integer"},
                            "file": {"type": "string"},
                            "fail": {"type": "string"},
                            "delay": {
                                "type": "object",
                                "properties": {
                                    "exact": {"type": "integer"},
                                    "min": {"type": "integer"},
                                    "max": {"type": "integer"}
                                }
                            },
                            "headers": {
                                "type": "object",
                                "additionalProperties": {"type": "string"}
                            },
                            "template": {"type": "boolean"}
                        }
                    },
                    "continue": {"type": "boolean"}
                }
            }
        },
        "system": {
            "type": "object",
            "properties": {
                "stores": {
                    "type": "object",
                    "additionalProperties": {
                        "type": "object",
                        "properties": {
                            "preloadFile": {"type": "string"},
                            "preloadData": {
                                "type": "object",
                                "additionalProperties": true
                            }
                        }
                    }
                },
                "xmlNamespaces": {
                    "type": "object",
                    "additionalProperties": {"type": "string"}
                }
            }
        }
    }
}

def validate_config_file(file_path):
    """Validate a single config file against the schema."""
    try:
        with open(file_path, 'r') as f:
            config = yaml.safe_load(f)
            validate(instance=config, schema=schema)
            print(f"✓ {file_path} - Valid")
            return True
    except ValidationError as e:
        print(f"✗ {file_path} - Invalid:")
        print(f"  Error: {e.message}")
        print(f"  Path: {' -> '.join(str(p) for p in e.path)}")
        return False
    except Exception as e:
        print(f"✗ {file_path} - Error reading/parsing file:")
        print(f"  {str(e)}")
        return False

def main():
    # Save schema to file
    with open('imposter-config-schema.json', 'w') as f:
        json.dump(schema, f, indent=2)
    print("Schema saved to imposter-config-schema.json")

    # Find and validate all config files
    examples_dir = Path('all-examples')
    if not examples_dir.exists():
        print("Error: all-examples directory not found")
        sys.exit(1)

    config_files = []
    for ext in ['-config.yaml', '-config.yml']:
        config_files.extend(examples_dir.rglob(f'*{ext}'))

    if not config_files:
        print("No config files found")
        sys.exit(1)

    print(f"\nValidating {len(config_files)} config files...")
    valid_count = sum(validate_config_file(f) for f in config_files)
    print(f"\nSummary: {valid_count}/{len(config_files)} files valid")

if __name__ == '__main__':
    main() 