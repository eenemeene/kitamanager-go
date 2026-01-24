.PHONY: build lint test clean ci \
	api-build api-run api-lint api-test-all api-test-unit api-test-integration api-test-contract api-test-fuzz api-test-coverage \
	web-install web-dev web-build web-lint web-lint-style web-format web-format-check web-type-check web-test web-test-coverage web-check-all \
	docs schema-docs swagger-docs docker-up docker-down docker-rebuild install-hooks uninstall-hooks pre-commit

# =============================================================================
# Combined targets (web + api)
# =============================================================================

# Build both web and api (web first)
build: web-build api-build

# Run linter for both web and api
lint: web-lint api-lint

# Run tests for both web and api
test: web-test api-test-unit

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html web/dist

# Run all CI checks locally
ci: lint test build
	@echo "All CI checks passed!"

# =============================================================================
# API targets
# =============================================================================

# Build the API application
api-build:
	go build -o bin/kitamanager-api ./cmd/api

# Run the API application locally
api-run:
	go run ./cmd/api

# Run API linter
api-lint:
	golangci-lint run ./...

# Run all API tests (unit, integration, contract - requires database)
api-test-all: api-test-unit api-test-integration api-test-contract

# Run API unit tests with race detection
api-test-unit:
	go test -v -race ./...

# Run API integration tests (requires database)
api-test-integration:
	go test -v -race -tags=integration ./internal/integration/...

# Run API contract tests (requires database)
api-test-contract:
	go test -v -tags=contract ./internal/contract/...

# Run API fuzz tests
api-test-fuzz:
	go test -fuzz=Fuzz -fuzztime=30s ./internal/models/...

# Run API tests with coverage report
api-test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# =============================================================================
# Web targets
# =============================================================================

# Install web dependencies
web-install:
	cd web && npm install

# Start web dev server
web-dev:
	cd web && npm run dev

# Build web for production
web-build:
	cd web && npm run build

# Lint web code (ESLint with accessibility checks)
web-lint:
	cd web && npm run lint

# Lint web styles (Stylelint)
web-lint-style:
	cd web && npm run lint:style

# Format web code (Prettier)
web-format:
	cd web && npm run format

# Check formatting without writing
web-format-check:
	cd web && npm run format:check

# Type-check web code
web-type-check:
	cd web && npm run type-check

# Run web unit tests
web-test:
	cd web && npm run test:run

# Run web tests with coverage
web-test-coverage:
	cd web && npm run test:coverage

# Run all web checks (type-check, lint, stylelint, format, tests)
web-check-all:
	cd web && npm run check-all

# =============================================================================
# Documentation targets
# =============================================================================

# Generate OpenAPI/Swagger documentation
swagger-docs:
	swag init -g cmd/api/main.go -o docs

# Update database schema documentation (requires running database)
schema-docs:
	tbls doc --force

# Generate all documentation
docs: swagger-docs schema-docs

# =============================================================================
# Docker targets
# =============================================================================

# Start docker containers (API + web + DB)
docker-up:
	docker compose up -d

# Stop docker containers
docker-down:
	docker compose down

# Rebuild and restart docker containers
docker-rebuild:
	docker compose up -d --build

# =============================================================================
# Git hooks targets
# =============================================================================

# Install pre-commit hooks
install-hooks:
	pre-commit install
	pre-commit install --hook-type commit-msg
	@echo "Pre-commit hooks installed."

# Uninstall pre-commit hooks
uninstall-hooks:
	pre-commit uninstall
	pre-commit uninstall --hook-type commit-msg
	@echo "Pre-commit hooks uninstalled."

# Run pre-commit on all files
pre-commit:
	pre-commit run --all-files
