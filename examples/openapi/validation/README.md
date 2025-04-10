# OpenAPI Validation Example

This example demonstrates how to use OpenAPI validation in Imposter. It includes a simple Pet Store API specification with validation rules for requests, and an Imposter configuration that enables validation with the "fail" behavior.

## Running the Example

Start Imposter in this directory (this will block the terminal):

```bash
imposter
```

Or from the parent directory:

```bash
imposter validation
```

Then open a new terminal to run the test curl commands. For convenience, you can also use the included test script:

```bash
./test-validation.sh
```

This script will run a series of valid and invalid requests to demonstrate the validation behaviors.

## Configuration

The `imposter-config.yaml` file enables request validation with the "fail" behavior:

```yaml
validation:
  request: fail
  response: log
```

This means:
- Invalid requests will be rejected with a 400 Bad Request response and detailed validation errors
- Response validation issues will be logged but will not affect the response

The OpenAPI specification (`petstore-validation.yaml`) includes examples for all schema elements, following best practices for OpenAPI documentation. These examples are used to generate response data automatically.

## Testing Request Validation

### Valid Requests

#### Create a valid pet:

```bash
curl -X POST http://localhost:8080/api/v1/pets \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "name": "Whiskers",
    "type": "cat",
    "age": 2,
    "vaccinated": true,
    "tags": ["playful", "friendly"]
  }'
```

Expected response: HTTP 201 Created with the created pet

#### Get all pets:

```bash
curl -H "Accept: application/json" http://localhost:8080/api/v1/pets
```

Expected response: HTTP 200 OK with a list of pets

#### Get a specific pet:

```bash
curl -H "Accept: application/json" http://localhost:8080/api/v1/pets/pet-1
```

Expected response: HTTP 200 OK with pet details

### Invalid Requests

#### Missing required field:

```bash
curl -X POST http://localhost:8080/api/v1/pets \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "name": "Whiskers"
  }'
```

Expected response: HTTP 400 Bad Request with validation errors about missing required 'type' field

#### Invalid enum value:

```bash
curl -X POST http://localhost:8080/api/v1/pets \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "name": "Jumbo",
    "type": "elephant"
  }'
```

Expected response: HTTP 400 Bad Request with validation errors about invalid value for 'type'

#### Invalid integer value:

```bash
curl -X POST http://localhost:8080/api/v1/pets \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "name": "Rex",
    "type": "dog",
    "age": -5
  }'
```

Expected response: HTTP 400 Bad Request with validation errors about 'age' being less than minimum

#### Invalid query parameter:

```bash
curl -H "Accept: application/json" "http://localhost:8080/api/v1/pets?limit=500"
```

Expected response: HTTP 400 Bad Request with validation errors about 'limit' exceeding maximum value

## Validation Behaviors

You can configure different validation behaviors in the `imposter-config.yaml`:

1. **Fail validation** (default):
   ```yaml
   validation:
     request: fail
   ```

2. **Log validation issues but allow the request**:
   ```yaml
   validation:
     request: log
   ```

3. **Ignore validation issues**:
   ```yaml
   validation:
     request: ignore
   ```

You can also use environment variables to configure the default behavior:
```bash
IMPOSTER_OPENAPI_VALIDATION_DEFAULT_BEHAVIOUR=log imposter
```