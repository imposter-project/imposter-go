#!/bin/bash

# Rate Limiting Test Scenarios
# This script demonstrates various load testing scenarios for the rate limiting example

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL=${IMPOSTER_URL:-"http://localhost:8080"}
HEY_OPTS=${HEY_OPTS:-""}

echo -e "${BLUE}Rate Limiting Test Scenarios${NC}"
echo -e "${BLUE}=============================${NC}"
echo ""
echo "Base URL: $BASE_URL"
echo ""

# Check if hey is installed
if ! command -v hey &> /dev/null; then
    echo -e "${RED}Error: 'hey' command not found${NC}"
    echo "Please install hey:"
    echo "  macOS: brew install hey"
    echo "  Linux: go install github.com/rakyll/hey@latest"
    echo "  Or download from: https://github.com/rakyll/hey/releases"
    exit 1
fi

# Check if server is running
echo -e "${YELLOW}Checking if Imposter server is running...${NC}"
if ! curl -s "$BASE_URL/health" > /dev/null; then
    echo -e "${RED}Error: Imposter server not responding at $BASE_URL${NC}"
    echo "Please start the server first:"
    echo "  go run cmd/imposter/main.go -configDir examples/rest/rate-limiting"
    exit 1
fi
echo -e "${GREEN}✓ Server is running${NC}"
echo ""

# Function to run a test scenario
run_test() {
    local name="$1"
    local description="$2"
    shift 2
    
    echo -e "${BLUE}Test: $name${NC}"
    echo -e "${YELLOW}Description: $description${NC}"
    echo -e "${YELLOW}Command: hey $*${NC}"
    echo ""
    
    hey "$@"
    
    echo ""
    echo -e "${GREEN}✓ Test completed${NC}"
    echo "----------------------------------------"
    echo ""
}

# Test 1: Light endpoint under normal load
run_test "Light Endpoint - Normal Load" \
    "Test light endpoint with moderate load (should mostly succeed)" \
    -n 50 -c 5 -m GET "$BASE_URL/api/light"

# Test 2: Light endpoint exceeding limit
run_test "Light Endpoint - Exceeding Limit" \
    "Test light endpoint with high concurrency (should see some 429s)" \
    -n 100 -c 15 -m GET "$BASE_URL/api/light"

# Test 3: Heavy endpoint progressive throttling
run_test "Heavy Endpoint - Progressive Throttling" \
    "Test heavy endpoint to trigger both throttling tiers" \
    -n 60 -c 8 -m GET "$BASE_URL/api/heavy"

# Test 4: Critical endpoint strict limits
run_test "Critical Endpoint - Strict Limits" \
    "Test critical endpoint with high concurrency (should see many 429s)" \
    -n 30 -c 10 -m POST \
    -H "Content-Type: application/json" \
    -d '{"operation": "critical_task", "priority": "high"}' \
    "$BASE_URL/api/critical"

# Test 5: Database endpoint circuit breaker
run_test "Database Endpoint - Circuit Breaker" \
    "Test database endpoint to trigger circuit breaker behavior" \
    -n 80 -c 12 -m GET "$BASE_URL/api/database"

# Test 6: Upload endpoint single slot
run_test "Upload Endpoint - Single Slot" \
    "Test upload endpoint with multiple concurrent requests (only 1 allowed)" \
    -n 20 -c 5 -m POST \
    -H "Content-Type: multipart/form-data" \
    -d '{"filename": "test.txt", "size": 1024}' \
    "$BASE_URL/api/upload"

# Test 7: Status endpoint (no rate limiting)
run_test "Status Endpoint - No Rate Limiting" \
    "Test status endpoint with high load (should all succeed)" \
    -n 100 -c 20 -m GET "$BASE_URL/api/status"

# Test 8: Mixed load simulation
echo -e "${BLUE}Test: Mixed Load Simulation${NC}"
echo -e "${YELLOW}Description: Simulate realistic mixed traffic across all endpoints${NC}"
echo ""

# Start background load on different endpoints
echo "Starting background load..."

# Light continuous load
hey -n 200 -c 3 -m GET "$BASE_URL/api/light" > /tmp/light_load.out 2>&1 &
LIGHT_PID=$!

# Heavy periodic spikes
hey -n 100 -c 6 -m GET "$BASE_URL/api/heavy" > /tmp/heavy_load.out 2>&1 &
HEAVY_PID=$!

# Critical sporadic requests
hey -n 40 -c 4 -m POST -H "Content-Type: application/json" -d '{"task": "mixed_test"}' "$BASE_URL/api/critical" > /tmp/critical_load.out 2>&1 &
CRITICAL_PID=$!

# Database queries
hey -n 80 -c 5 -m GET "$BASE_URL/api/database" > /tmp/database_load.out 2>&1 &
DATABASE_PID=$!

# Status monitoring (should always work)
hey -n 50 -c 2 -m GET "$BASE_URL/api/status" > /tmp/status_load.out 2>&1 &
STATUS_PID=$!

echo "Waiting for all tests to complete..."

# Wait for all background jobs
wait $LIGHT_PID
wait $HEAVY_PID  
wait $CRITICAL_PID
wait $DATABASE_PID
wait $STATUS_PID

echo ""
echo -e "${GREEN}Mixed load test completed. Results:${NC}"
echo ""

# Show summary of each endpoint
echo -e "${YELLOW}Light endpoint results:${NC}"
tail -n 20 /tmp/light_load.out
echo ""

echo -e "${YELLOW}Heavy endpoint results:${NC}"
tail -n 20 /tmp/heavy_load.out
echo ""

echo -e "${YELLOW}Critical endpoint results:${NC}"
tail -n 20 /tmp/critical_load.out
echo ""

echo -e "${YELLOW}Database endpoint results:${NC}"
tail -n 20 /tmp/database_load.out
echo ""

echo -e "${YELLOW}Status endpoint results:${NC}"
tail -n 20 /tmp/status_load.out
echo ""

# Cleanup temp files
rm -f /tmp/*_load.out

echo "----------------------------------------"
echo ""

# Test 9: Burst test
run_test "Burst Test - All Endpoints" \
    "Quick burst test across all endpoints to see rate limiting in action" \
    -n 20 -c 20 -m GET "$BASE_URL/api/heavy"

# Final status check
echo -e "${BLUE}Final Status Check${NC}"
echo -e "${YELLOW}Getting current server status after load testing...${NC}"
echo ""

curl -s "$BASE_URL/api/status" | jq . || curl -s "$BASE_URL/api/status"

echo ""
echo -e "${GREEN}✓ All rate limiting tests completed successfully!${NC}"
echo ""
echo -e "${YELLOW}Key observations to look for:${NC}"
echo "• Light endpoint: Should handle 10 concurrent, 429 beyond that"
echo "• Heavy endpoint: Throttling at 3 concurrent, 503 at 5+ concurrent"  
echo "• Critical endpoint: Strict 2 concurrent limit with 429 responses"
echo "• Database endpoint: Throttling at 5, circuit breaker at 8+"
echo "• Upload endpoint: Only 1 concurrent allowed"
echo "• Status endpoint: No rate limiting, should always succeed"
echo ""
echo -e "${YELLOW}Rate limiting logs should show:${NC}"
echo "• 'rate limit exceeded' messages for blocked requests"
echo "• 'rate limit applied' messages when limits trigger"
echo "• TTL cleanup messages after time passes"
echo ""