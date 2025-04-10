#!/bin/bash
# Test script for OpenAPI validation example
# Run this after starting the Imposter server

echo "Testing OpenAPI validation example"
echo "=================================="
echo ""

# Function to make curl requests and display results
make_request() {
  echo "$1"
  echo "-------------------------------------------------"
  echo "> $2"
  echo ""
  eval "$2"
  echo ""
  echo ""
}

# Test valid requests
make_request "1. Valid request - Get all pets" "curl -s -H 'Accept: application/json' http://localhost:8080/api/v1/pets | head -10"

make_request "2. Valid request - Get specific pet" "curl -s -H 'Accept: application/json' http://localhost:8080/api/v1/pets/pet-1"

make_request "3. Valid request - Create a valid pet" \
  "curl -s -X POST http://localhost:8080/api/v1/pets \\
    -H 'Content-Type: application/json' \\
    -H 'Accept: application/json' \\
    -d '{\"name\": \"Whiskers\", \"type\": \"cat\", \"age\": 2, \"vaccinated\": true, \"tags\": [\"playful\", \"friendly\"]}'"

# Test invalid requests
make_request "4. Invalid request - Missing required field" \
  "curl -s -X POST http://localhost:8080/api/v1/pets \\
    -H 'Content-Type: application/json' \\
    -H 'Accept: application/json' \\
    -d '{\"name\": \"Whiskers\"}'"

make_request "5. Invalid request - Invalid enum value" \
  "curl -s -X POST http://localhost:8080/api/v1/pets \\
    -H 'Content-Type: application/json' \\
    -H 'Accept: application/json' \\
    -d '{\"name\": \"Jumbo\", \"type\": \"elephant\"}'"

make_request "6. Invalid request - Negative age" \
  "curl -s -X POST http://localhost:8080/api/v1/pets \\
    -H 'Content-Type: application/json' \\
    -H 'Accept: application/json' \\
    -d '{\"name\": \"Rex\", \"type\": \"dog\", \"age\": -5}'"

make_request "7. Invalid request - Invalid query parameter" \
  "curl -s -H 'Accept: application/json' \"http://localhost:8080/api/v1/pets?limit=500\""

echo "Testing complete. Start the server with different validation behaviors to see how responses change."
echo "For example: IMPOSTER_OPENAPI_VALIDATION_DEFAULT_BEHAVIOUR=log imposter"