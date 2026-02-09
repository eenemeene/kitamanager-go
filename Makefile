.PHONY: build lint test clean ci dev \
	api-build api-run api-lint api-test-all api-test-unit api-test-integration api-test-contract api-test-fuzz api-test-coverage \
	web-install web-dev web-build web-lint web-format web-format-check web-type-check web-test web-test-coverage web-test-e2e web-test-e2e-fresh web-test-e2e-demo \
	docs schema-docs swagger-docs docker-up docker-down docker-rebuild docker-reset install-hooks uninstall-hooks pre-commit

# =============================================================================
# Version info
# =============================================================================
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
VERSION_PKG := github.com/eenemeene/kitamanager-go/internal/version
LDFLAGS := -ldflags "-X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).BuildTime=$(BUILD_TIME)"

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
	rm -rf bin/ coverage.out coverage.html frontend/.next

# Run all CI checks locally
ci: lint test build
	@echo "All CI checks passed!"

# Start full development environment (database + API + web with hot reload)
# Prerequisites: Docker for database, Go and Node.js installed
# Web UI will be available at http://localhost:3000 with hot reload
# API will be available at http://localhost:8080
dev:
	@echo "Starting development environment..."
	@echo "1. Starting database..."
	@docker compose up -d db
	@echo "2. Waiting for database to be healthy..."
	@until docker compose exec -T db pg_isready -U kitamanager > /dev/null 2>&1; do \
		echo "   Waiting for PostgreSQL..."; \
		sleep 1; \
	done
	@echo "   Database is ready!"
	@echo "3. Starting API server in background..."
	@DATABASE_URL="postgres://kitamanager:kitamanager@localhost:5432/kitamanager?sslmode=disable" \
		SEED_ADMIN_EMAIL=admin@example.com \
		SEED_ADMIN_PASSWORD=supersecret \
		SEED_ADMIN_NAME=admin \
		SEED_RBAC_POLICIES=true \
		SEED_TEST_DATA=true \
		GOVERNMENT_FUNDING_SEED_PATH=configs/government-fundings/berlin.yaml \
		GOVERNMENT_FUNDING_SEED_NAME=Berlin \
		CORS_ALLOW_ORIGINS="http://localhost:3000,http://localhost:3001,http://localhost:8080" \
		CORS_ALLOW_CREDENTIALS=true \
		LOGIN_RATE_LIMIT_PER_MINUTE=0 \
		./bin/kitamanager-api > /tmp/kitamanager-api.log 2>&1 & echo $$! > /tmp/kitamanager-api.pid
	@echo "   Waiting for API to be healthy..."
	@until curl -sf http://localhost:8080/api/v1/health > /dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "   API is ready!"
	@echo "4. Starting web dev server (Ctrl+C to stop all)..."
	@echo ""
	@echo "================================================"
	@echo "  Web UI: http://localhost:3000 (hot reload)"
	@echo "  API:    http://localhost:8080"
	@echo "  Login:  admin@example.com / supersecret"
	@echo "================================================"
	@echo ""
	@trap 'kill $$(cat /tmp/kitamanager-api.pid) 2>/dev/null; docker compose stop db' EXIT; \
		cd frontend && npm run dev

# =============================================================================
# API targets
# =============================================================================

# Build the API application
api-build:
	go build $(LDFLAGS) -o bin/kitamanager-api ./cmd/api

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

# Run API fuzz tests (each fuzz test must be run separately)
api-test-fuzz:
	go test -fuzz=FuzzPeriodOverlaps -fuzztime=30s ./internal/models/...
	go test -fuzz=FuzzPeriodIsActiveOn -fuzztime=30s ./internal/models/...

# Run API tests with coverage report
api-test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# =============================================================================
# Web targets (Next.js frontend)
# =============================================================================

# Install web dependencies
web-install:
	cd frontend && npm install

# Start web dev server
web-dev:
	cd frontend && npm run dev

# Build web for production
web-build:
	cd frontend && npm run build

# Lint web code
web-lint:
	cd frontend && npm run lint

# Format web code (Prettier)
web-format:
	cd frontend && npm run format

# Check formatting without writing
web-format-check:
	cd frontend && npm run format:check

# Type-check web code
web-type-check:
	cd frontend && npm run type-check

# Run web unit tests
web-test:
	cd frontend && npm run test

# Run web tests with coverage
web-test-coverage:
	cd frontend && npm run test:coverage

# Run web E2E tests (requires dev server running or will start it)
web-test-e2e:
	cd frontend && npm run test:e2e

# Run web E2E tests with a fresh database (resets all data first)
web-test-e2e-fresh:
	@echo "Stopping any running API server..."
	@-kill $$(cat /tmp/kitamanager-api.pid 2>/dev/null) 2>/dev/null || true
	@rm -f /tmp/kitamanager-api.pid
	docker compose down -v
	@echo "Database reset. Starting E2E tests with fresh database..."
	cd frontend && npm run test:e2e

# Run web E2E tests with browser visible
web-test-e2e-headed:
	cd frontend && npm run test:e2e:headed

# Run web E2E tests with browser visible and slow motion (for human watching)
# Uses Chromium for better video recording support
web-test-e2e-demo:
	cd frontend && SLOWMO=500 VIDEO=1 npx playwright test --headed --project=chromium

# Install Playwright browsers
web-playwright-install:
	cd frontend && npx playwright install --with-deps

# =============================================================================
# Documentation targets
# =============================================================================

# Generate OpenAPI/Swagger documentation
swagger-docs:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

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

# Reset database (removes all data)
docker-reset:
	docker compose down -v
	@echo "Database volume removed. Run 'make dev' to start fresh."

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
