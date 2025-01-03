#!/usr/bin/env python3

import json
import os
import sys
import argparse
from pathlib import Path
import yaml
from jsonschema import validate, ValidationError, Draft7Validator

def load_schema():
    """Load the JSON schema from file."""
    schema_file = Path(__file__).parent / 'imposter-config-schema.json'
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
    args = parser.parse_args()

    # Load schema
    schema = load_schema()

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

    print(f"\nValidating {len(config_files)} config files...")
    valid_count = sum(validate_config_file(f, schema) for f in sorted(config_files))
    print(f"\nSummary: {valid_count}/{len(config_files)} files valid")

if __name__ == '__main__':
    main() 