# Rate Limiting Example Makefile
# Provides convenient commands for testing rate limiting functionality

.PHONY: help start start-redis start-dynamodb start-multi stop test test-light test-heavy test-critical test-all clean logs status

# Default target
help:
	@echo "Rate Limiting Example Commands"
	@echo "=============================="
	@echo ""
	@echo "Basic Usage:"
	@echo "  make start          Start Imposter with in-memory store"
	@echo "  make start-redis    Start Imposter with Redis store"
	@echo "  make start-dynamodb Start Imposter with DynamoDB store"
	@echo "  make start-multi    Start multiple instances with Redis"
	@echo "  make stop           Stop all services"
	@echo ""
	@echo "Testing:"
	@echo "  make test           Run all test scenarios"
	@echo "  make test-light     Test light endpoint"
	@echo "  make test-heavy     Test heavy endpoint"
	@echo "  make test-critical  Test critical endpoint"
	@echo "  make test-upload    Test upload endpoint"
	@echo "  make test-mixed     Run mixed load test"
	@echo ""
	@echo "Monitoring:"
	@echo "  make status         Check server status"
	@echo "  make logs           Show server logs"
	@echo "  make logs-redis     Show Redis logs"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean          Clean up Docker resources"
	@echo "  make install-hey    Install hey load testing tool"

# Start services
start:
	@echo "Starting Imposter with in-memory store..."
	docker-compose --profile inmemory up -d
	@echo "Waiting for service to be ready..."
	@sleep 5
	@make status

start-redis:
	@echo "Starting Imposter with Redis store..."
	docker-compose --profile redis up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@make status

start-dynamodb:
	@echo "Starting Imposter with DynamoDB store..."
	docker-compose --profile dynamodb up -d
	@echo "Waiting for services to be ready..."
	@sleep 15
	@make status

start-multi:
	@echo "Starting multiple Imposter instances with Redis..."
	docker-compose --profile multi-redis up -d
	@echo "Waiting for services to be ready..."
	@sleep 15
	@echo "Instance 1 status:"
	@curl -s http://localhost:8091/api/status | jq '.status // "Could not connect"' || echo "Instance 1 not ready"
	@echo "Instance 2 status:"
	@curl -s http://localhost:8092/api/status | jq '.status // "Could not connect"' || echo "Instance 2 not ready"

# Stop services
stop:
	@echo "Stopping all services..."
	docker-compose --profile inmemory --profile redis --profile dynamodb --profile multi-redis down

# Test scenarios
test: install-hey
	@echo "Running all test scenarios..."
	./test-scenarios.sh

test-light: install-hey
	@echo "Testing light endpoint (limit: 10 concurrent)..."
	hey -n 50 -c 15 -m GET http://localhost:8080/api/light

test-heavy: install-hey
	@echo "Testing heavy endpoint (throttling: 3, rejection: 5)..."
	hey -n 40 -c 8 -m GET http://localhost:8080/api/heavy

test-critical: install-hey
	@echo "Testing critical endpoint (limit: 2 concurrent)..."
	hey -n 20 -c 6 -m POST -H "Content-Type: application/json" -d '{"test": "critical"}' http://localhost:8080/api/critical

test-upload: install-hey
	@echo "Testing upload endpoint (limit: 1 concurrent)..."
	hey -n 10 -c 4 -m POST -H "Content-Type: application/json" -d '{"file": "test.txt"}' http://localhost:8080/api/upload

test-database: install-hey
	@echo "Testing database endpoint (throttling: 5, circuit breaker: 8)..."
	hey -n 60 -c 10 -m GET http://localhost:8080/api/database

test-mixed: install-hey
	@echo "Running mixed load test..."
	@echo "Starting background loads..."
	@hey -n 100 -c 3 -m GET http://localhost:8080/api/light > /tmp/light.out 2>&1 &
	@hey -n 50 -c 6 -m GET http://localhost:8080/api/heavy > /tmp/heavy.out 2>&1 &
	@hey -n 30 -c 4 -m POST -H "Content-Type: application/json" -d '{"mixed": "test"}' http://localhost:8080/api/critical > /tmp/critical.out 2>&1 &
	@echo "Waiting for tests to complete..."
	@sleep 20
	@echo "Light endpoint results:"
	@tail -n 10 /tmp/light.out
	@echo "Heavy endpoint results:"
	@tail -n 10 /tmp/heavy.out
	@echo "Critical endpoint results:"
	@tail -n 10 /tmp/critical.out
	@rm -f /tmp/*.out

# Test multi-instance rate limiting
test-multi: install-hey
	@echo "Testing distributed rate limiting across multiple instances..."
	@echo "Instance 1 (port 8091) - Heavy endpoint:"
	@hey -n 30 -c 5 -m GET http://localhost:8091/api/heavy &
	@echo "Instance 2 (port 8092) - Heavy endpoint:"
	@hey -n 30 -c 5 -m GET http://localhost:8092/api/heavy &
	@echo "Waiting for both tests to complete..."
	@wait

# Monitoring
status:
	@echo "Checking server status..."
	@curl -s http://localhost:8080/api/status 2>/dev/null | jq . || curl -s http://localhost:8080/api/status || echo "Server not responding"

logs:
	docker-compose logs -f imposter-inmemory imposter-redis imposter-dynamodb

logs-redis:
	docker-compose logs -f redis

logs-dynamodb:
	docker-compose logs -f dynamodb

# Development helpers
dev-start:
	@echo "Starting development server..."
	cd ../../.. && go run cmd/imposter/main.go -configDir examples/rest/rate-limiting

dev-test: install-hey
	@echo "Testing development server..."
	IMPOSTER_URL=http://localhost:8080 ./test-scenarios.sh

# Utilities
clean:
	@echo "Cleaning up Docker resources..."
	docker-compose --profile inmemory --profile redis --profile dynamodb --profile multi-redis down -v
	docker system prune -f

install-hey:
	@if ! command -v hey >/dev/null 2>&1; then \
		echo "Installing hey load testing tool..."; \
		if command -v brew >/dev/null 2>&1; then \
			brew install hey; \
		elif command -v go >/dev/null 2>&1; then \
			go install github.com/rakyll/hey@latest; \
		else \
			echo "Please install hey manually: https://github.com/rakyll/hey/releases"; \
			exit 1; \
		fi; \
	fi

# Performance benchmarks
benchmark: install-hey
	@echo "Running performance benchmarks..."
	@echo "==================================="
	@echo "Light endpoint (10 concurrent limit):"
	@hey -n 200 -c 5 -q 10 -m GET http://localhost:8080/api/light | grep -E "(Requests/sec|Total:|Status code)"
	@echo ""
	@echo "Heavy endpoint (3/5 concurrent limits):"
	@hey -n 100 -c 4 -q 5 -m GET http://localhost:8080/api/heavy | grep -E "(Requests/sec|Total:|Status code)"
	@echo ""
	@echo "Status endpoint (no limits):"
	@hey -n 500 -c 20 -q 50 -m GET http://localhost:8080/api/status | grep -E "(Requests/sec|Total:|Status code)"

# Docker helpers
build:
	@echo "Building Imposter Docker image..."
	cd ../../.. && docker build -t imposter-go:latest .

push:
	@echo "Note: Add your registry and push logic here"

# Health checks
health:
	@echo "Health check results:"
	@echo "===================="
	@curl -s http://localhost:8080/health | jq . || echo "Health endpoint failed"
	@echo ""
	@curl -s http://localhost:8080/api/status | jq '.status' || echo "Status endpoint failed"

# Configuration validation
validate:
	@echo "Validating configuration..."
	@if command -v yq >/dev/null 2>&1; then \
		yq eval . imposter-config.yaml >/dev/null && echo "✓ Configuration is valid YAML"; \
	else \
		echo "Install yq to validate YAML configuration"; \
	fi

# Show configuration
show-config:
	@echo "Current configuration:"
	@echo "====================="
	@cat imposter-config.yaml