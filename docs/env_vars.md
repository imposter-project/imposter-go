# Environment Variables

This document lists all environment variables that Imposter supports, their purposes, default values, and examples.

## Core Configuration

| Variable | Purpose | Default Value | Example |
|----------|---------|---------------|---------|
| `IMPOSTER_PORT` | HTTP server port | `"8080"` | `IMPOSTER_PORT=3000` |
| `IMPOSTER_SERVER_URL` | Server base URL | `http://localhost:{PORT}` | `IMPOSTER_SERVER_URL=https://api.example.com` |
| `IMPOSTER_CONFIG_DIR` | Configuration directory path | Current directory | `IMPOSTER_CONFIG_DIR=/app/config` |
| `IMPOSTER_CONFIG_SCAN_RECURSIVE` | Enable recursive config scanning | `false` | `IMPOSTER_CONFIG_SCAN_RECURSIVE=true` |
| `IMPOSTER_AUTO_BASE_PATH` | Auto-generate base paths from directory structure | `false` | `IMPOSTER_AUTO_BASE_PATH=true` |
| `IMPOSTER_SUPPORT_LEGACY_CONFIG` | Enable legacy configuration format support | `false` | `IMPOSTER_SUPPORT_LEGACY_CONFIG=true` |

## Logging

| Variable | Purpose | Default Value | Example |
|----------|---------|---------------|---------|
| `IMPOSTER_LOG_LEVEL` | Logging level | `"DEBUG"` | `IMPOSTER_LOG_LEVEL=INFO` |

## Store Configuration

| Variable | Purpose | Default Value | Example |
|----------|---------|---------------|---------|
| `IMPOSTER_STORE_DRIVER` | Store backend driver | In-memory | `IMPOSTER_STORE_DRIVER=store-dynamodb` |
| `IMPOSTER_STORE_KEY_PREFIX` | Prefix for all store keys | No prefix | `IMPOSTER_STORE_KEY_PREFIX=imposter:` |

## DynamoDB Store

| Variable | Purpose | Default Value | Example |
|----------|---------|---------------|---------|
| `IMPOSTER_DYNAMODB_REGION` | AWS region for DynamoDB | `AWS_REGION` value | `IMPOSTER_DYNAMODB_REGION=us-west-2` |
| `IMPOSTER_DYNAMODB_TABLE` | DynamoDB table name | No default (required) | `IMPOSTER_DYNAMODB_TABLE=imposter-data` |
| `IMPOSTER_DYNAMODB_TTL` | TTL for DynamoDB items (seconds) | No TTL (-1) | `IMPOSTER_DYNAMODB_TTL=3600` |
| `IMPOSTER_DYNAMODB_TTL_ATTRIBUTE` | DynamoDB TTL attribute name | `"ttl"` | `IMPOSTER_DYNAMODB_TTL_ATTRIBUTE=expires_at` |

## Redis Store

| Variable | Purpose | Default Value | Example |
|----------|---------|---------------|---------|
| `REDIS_ADDR` | Redis server address | `"localhost:6379"` | `REDIS_ADDR=redis.example.com:6379` |
| `REDIS_PASSWORD` | Redis authentication password | No password | `REDIS_PASSWORD=secretpassword` |
| `IMPOSTER_STORE_REDIS_EXPIRY` | Redis key expiration duration | No expiration (-1) | `IMPOSTER_STORE_REDIS_EXPIRY=30m` |

## OpenAPI Validation

| Variable | Purpose | Default Value | Example |
|----------|---------|---------------|---------|
| `IMPOSTER_OPENAPI_VALIDATION_DEFAULT_BEHAVIOUR` | Default validation behavior | Context-dependent | `IMPOSTER_OPENAPI_VALIDATION_DEFAULT_BEHAVIOUR=fail` |

## AWS Integration

| Variable | Purpose | Default Value | Example |
|----------|---------|---------------|---------|
| `AWS_LAMBDA_FUNCTION_NAME` | AWS Lambda function name (auto-detected) | Not set | `AWS_LAMBDA_FUNCTION_NAME=my-function` |
| `AWS_REGION` | AWS region for SDK operations | Not set | `AWS_REGION=eu-west-1` |
| `AWS_ACCESS_KEY_ID` | AWS access key ID | Not set | `AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE` |
| `AWS_SECRET_ACCESS_KEY` | AWS secret access key | Not set | `AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY` |
| `AWS_DEFAULT_REGION` | AWS default region | Not set | `AWS_DEFAULT_REGION=us-east-1` |
| `AWS_ENDPOINT_URL` | Custom AWS endpoint URL | Not set | `AWS_ENDPOINT_URL=http://localhost:4566` |

## Template Variable Substitution

Imposter supports environment variable substitution in configuration files using the `${env.VARIABLE_NAME}` syntax:

```yaml
# In configuration files
server:
  port: ${env.PORT:-8080}
  host: ${env.HOST:-localhost}
database:
  url: ${env.DATABASE_URL}
```

| Syntax | Purpose | Example |
|--------|---------|---------|
| `${env.VAR_NAME}` | Substitute environment variable | `${env.DATABASE_URL}` |
| `${env.VAR_NAME:-default}` | Substitute with default if not set | `${env.PORT:-8080}` |

## Usage Examples

### Basic Setup
```bash
export IMPOSTER_PORT=3000
export IMPOSTER_CONFIG_DIR=/app/configs
export IMPOSTER_LOG_LEVEL=INFO
imposter
```

### DynamoDB Backend
```bash
export IMPOSTER_STORE_DRIVER=store-dynamodb
export IMPOSTER_DYNAMODB_REGION=us-west-2
export IMPOSTER_DYNAMODB_TABLE=imposter-data
export IMPOSTER_DYNAMODB_TTL=3600
imposter
```

### Redis Backend
```bash
export IMPOSTER_STORE_DRIVER=store-redis
export REDIS_ADDR=redis.example.com:6379
export REDIS_PASSWORD=mypassword
export IMPOSTER_STORE_REDIS_EXPIRY=1h
imposter
```

### Rate Limiting Configuration
```bash
export IMPOSTER_RATE_LIMITER_TTL=600
export IMPOSTER_RATE_LIMITER_AUTO_CLEANUP=true
imposter
```

### AWS Lambda Deployment
```bash
export AWS_REGION=eu-west-1
export IMPOSTER_CONFIG_DIR=/var/task/config
export IMPOSTER_STORE_DRIVER=store-dynamodb
export IMPOSTER_DYNAMODB_TABLE=imposter-lambda-data
export IMPOSTER_DYNAMODB_TTL=300
```

## Notes

1. **Boolean Values**: For boolean environment variables, use `"true"`, `"1"`, `"false"`, or `"0"`
2. **Duration Values**: Duration strings can use suffixes like `s` (seconds), `m` (minutes), `h` (hours)
3. **AWS Credentials**: Use IAM roles when possible instead of hardcoding access keys
4. **Lambda Auto-Detection**: When `AWS_LAMBDA_FUNCTION_NAME` is present, Imposter automatically runs in Lambda mode
5. **Store Driver Values**: 
   - `"store-dynamodb"` for DynamoDB backend
   - `"store-redis"` for Redis backend  
   - Empty or any other value for in-memory backend
6. **TTL Values**: Use `-1` or empty string to disable TTL/expiration
7. **Log Levels**: Valid values are `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`
