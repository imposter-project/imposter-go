#!/usr/bin/env python3

import json
import os
import sys
import argparse
from pathlib import Path
import yaml
from jsonschema import validate, ValidationError, Draft7Validator, RefResolver
from urllib.parse import urlparse

def load_schema(schema_path=None, schema_type='current'):
    """Load the JSON schema from file."""
    schema_dir = Path(__file__).parent.resolve()
    
    if schema_path:
        schema_file = Path(schema_path).resolve()
    else:
        # Use either current or legacy format schema based on schema_type
        if schema_type == 'legacy':
            schema_file = schema_dir / 'legacy-format-schema.json'
        else:
            schema_file = schema_dir / 'current-format-schema.json'
        
        if not schema_file.exists():
            print(f"Error: Schema file not found at {schema_file}")
            sys.exit(1)
    
    try:
        with open(schema_file, 'r') as f:
            return json.load(f)
    except Exception as e:
        print(f"Error loading schema file: {e}")
        sys.exit(1)

def validate_config_file(file_path, schema):
    """Validate a single config file against the schema."""
    try:
        with open(file_path, 'r') as f:
            # Handle multi-document YAML files
            documents = list(yaml.safe_load_all(f))
            
            # Create a resolver that can handle references to other schema files
            schema_dir = Path(__file__).parent.resolve()
            
            # Load shared definitions schema
            shared_schema = json.loads((schema_dir / 'shared-definitions.json').read_text())
            
            # Create store with relative paths
            store = {
                'shared-definitions.json': shared_schema
            }
            
            # Create a resolver that handles relative paths
            def custom_uri_handler(uri):
                parsed = urlparse(uri)
                if parsed.scheme == '':  # Relative path
                    path = parsed.path.lstrip('/')  # Remove leading slash if present
                    return store.get(path)
                return None
            
            resolver = RefResolver.from_schema(
                schema,
                store=store,
                handlers={'': custom_uri_handler}
            )
            
            validator = Draft7Validator(schema, resolver=resolver)
            
            # Validate each document
            all_valid = True
            for i, config in enumerate(documents):
                if config is None:  # Skip empty documents
                    continue
                try:
                    validator.validate(config)
                except ValidationError as e:
                    print(f"✗ {file_path} - Document {i+1} Invalid:")
                    print(f"  Error: {e.message}")
                    print(f"  Path: {' -> '.join(str(p) for p in e.path)}")
                    all_valid = False
            
            if all_valid:
                doc_count = len([d for d in documents if d is not None])
                if doc_count > 1:
                    print(f"✓ {file_path} - Valid ({doc_count} documents)")
                else:
                    print(f"✓ {file_path} - Valid")
                return True
            return False
            
    except ValidationError as e:
        print(f"✗ {file_path} - Invalid:")
        print(f"  Error: {e.message}")
        print(f"  Path: {' -> '.join(str(p) for p in e.path)}")
        return False
    except Exception as e:
        print(f"✗ {file_path} - Error reading/parsing file:")
        print(f"  {str(e)}")
        return False

def find_config_files(root_dir):
    """Find all config files in the given directory."""
    config_files = []
    for ext in ['-config.yaml', '-config.yml']:
        config_files.extend(Path(root_dir).rglob(f'*{ext}'))
    return config_files

def main():
    parser = argparse.ArgumentParser(description='Validate Imposter configuration files against JSON schema.')
    parser.add_argument('directory', help='Directory containing config files to validate')
    parser.add_argument('--schema', help='Path to custom schema file (optional)')
    parser.add_argument('--schema-type', choices=['current', 'legacy'], default='current',
                      help='Type of schema to validate against (default: current)')
    args = parser.parse_args()

    # Load schema
    schema = load_schema(args.schema, args.schema_type)

    # Convert relative path to absolute path
    search_dir = Path(args.directory).resolve()
    if not search_dir.exists():
        print(f"Error: Directory not found: {search_dir}")
        sys.exit(1)

    # Find and validate all config files
    config_files = find_config_files(search_dir)
    if not config_files:
        print("No config files found")
        sys.exit(1)

    print(f"\nValidating {len(config_files)} config files against {args.schema_type} schema...")
    valid_count = sum(validate_config_file(f, schema) for f in sorted(config_files))
    print(f"\nSummary: {valid_count}/{len(config_files)} files valid")

if __name__ == '__main__':
    main() 