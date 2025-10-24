# Exchange Simulator Go - Makefile

.PHONY: help test test-unit test-integration test-all build clean lint

# Load environment variables from .env file if it exists
ifneq (,$(wildcard .env))
	include .env
	export
endif

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Environment setup
check-env: ## Check if .env file exists
	@if [ ! -f .env ]; then \
		echo "Warning: .env file not found. Copy .env.example to .env and configure."; \
		echo "  cp .env.example .env"; \
		exit 1; \
	fi
	@echo ".env file found ✓"

# Test targets
test: test-unit ## Run unit tests (default)

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi && go test -tags=unit ./internal/... -v

test-integration: check-env ## Run integration tests only (requires .env)
	@echo "Running integration tests..."
	@set -a && . ./.env && set +a && go test -tags=integration ./internal/... -v

test-all: check-env ## Run all tests (unit + integration)
	@echo "Running all tests..."
	@set -a && . ./.env && set +a && go test -tags="unit integration" ./internal/... -v

test-short: ## Run tests in short mode (skip slow tests)
	@echo "Running tests in short mode..."
	go test -tags=unit ./internal/... -short -v

# Build targets
build: ## Build the exchange simulator binary
	@echo "Building exchange simulator..."
	go build -o exchange-simulator ./cmd/server

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f exchange-simulator server
	go clean -testcache

# Development targets
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Info targets
test-list: ## List all available tests
	@echo "Unit tests:"
	@go test -tags=unit ./internal/... -list=. 2>/dev/null || echo "  (Run 'make test-unit' to see unit tests)"
	@echo ""
	@echo "Integration tests:"
	@go test -tags=integration ./internal/... -list=. 2>/dev/null || echo "  (Run 'make test-integration' to see integration tests)"

test-files: ## Show test files
	@echo "Test files in exchange-simulator-go:"
	@find . -name "*_test.go" -exec echo "  {}" \;

# Status check
status: ## Check current test status
	@echo "=== Exchange Simulator Go Test Status ==="
	@echo ""
	@echo "Unit Tests (tags=unit):"
	@go test -tags=unit ./internal/... -v 2>&1 | grep -E "(PASS|FAIL|SKIP|===)" | head -10 || echo "  No unit tests found"
	@echo ""
	@echo "Integration Tests (tags=integration):"
	@go test -tags=integration ./internal/... -v 2>&1 | grep -E "(PASS|FAIL|SKIP|===)" | head -10 || echo "  No integration tests found"
