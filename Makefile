.PHONY: build lint test clean ci dev dev-fresh \
	api-build api-run api-lint api-test-all api-test-unit api-test-integration api-test-contract api-test-fuzz api-test-coverage api-test-backup api-test-race \
	web-install web-dev web-build web-lint web-format web-format-check web-type-check web-test web-test-coverage web-test-e2e web-test-e2e-fresh web-test-e2e-demo web-test-e2e-headed web-playwright-install screenshots \
	docs schema-docs swagger-docs swagger-check api-types api-types-check docker-up docker-down docker-rebuild docker-reset install-hooks uninstall-hooks pre-commit \
	report-pdf-build report-pdf

# =============================================================================
# Version info
# =============================================================================
GIT_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
VERSION_PKG := github.com/eenemeene/kitamanager-go/internal/version
LDFLAGS := -ldflags "-X $(VERSION_PKG).GitVersion=$(GIT_VERSION) -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).BuildTime=$(BUILD_TIME)"

# =============================================================================
# Combined targets (web + api)
# =============================================================================

# Build both web and api (installs deps, then builds)
build: web-install web-build api-build

# Run linter for both web and api
lint: web-lint api-lint

# Run tests for both web and api
test: web-test api-test-unit

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html frontend/.next

# Fast local gate: lint, unit tests and both builds. NOT the same as CI —
# GitHub additionally runs web-type-check, web-format-check, swagger-check,
# api-types-check, api-test-integration, api-test-contract, api-test-fuzz and
# the E2E suite. Green here does not mean green there; `make pre-commit` is the
# closer approximation.
ci: lint test build
	@echo "Local checks passed (lint, unit tests, builds)."
	@echo "This is not the full CI suite — see the comment above this target."

# Reset everything and start fresh development environment
dev-fresh: docker-reset clean dev

# Start full development environment (database + API + web with hot reload)
# Prerequisites: Docker for database, Go and Node.js installed
# Web UI will be available at http://localhost:3000 with hot reload
# API will be available at http://localhost:8080
dev: api-build web-install
	@echo "Starting development environment..."
	@if [ ! -f .env ]; then \
		echo "0. No .env found — creating from .env.dev.example"; \
		cp .env.dev.example .env; \
		CSRF=$$(openssl rand -hex 32); \
		TOTP=$$(openssl rand -hex 32); \
		sed -i.bak "s|^CSRF_HMAC_KEY=.*|CSRF_HMAC_KEY=$$CSRF|; s|^TOTP_ENCRYPTION_KEY=.*|TOTP_ENCRYPTION_KEY=$$TOTP|" .env && rm .env.bak; \
		echo "   Generated dev-only CSRF_HMAC_KEY and TOTP_ENCRYPTION_KEY in .env"; \
	fi
	@echo "1. Starting database..."
	@docker compose up -d db
	@echo "2. Waiting for database to be healthy..."
	@until docker compose exec -T db pg_isready -U kitamanager > /dev/null 2>&1; do \
		echo "   Waiting for PostgreSQL..."; \
		sleep 1; \
	done
	@echo "   Database is ready!"
	@echo "3. Starting API server in background..."
	@DB_HOST=localhost \
		DB_PORT=5432 \
		DB_USER=kitamanager \
		DB_PASSWORD=kitamanager \
		DB_NAME=kitamanager \
		DB_SSLMODE=disable \
		CSRF_HMAC_KEY=make-dev-csrf-hmac-key-eenemeene-2026-not-for-production-use \
		TOTP_ENCRYPTION_KEY=deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef \
		WEBAUTHN_RP_ID=localhost \
		WEBAUTHN_RP_NAME="KitaManager (dev)" \
		WEBAUTHN_ORIGINS="http://localhost:3000,http://localhost:8080" \
		SECURE_COOKIES=false \
		SEED_ADMIN_EMAIL=admin@example.com \
		SEED_ADMIN_PASSWORD=supersecret \
		SEED_ADMIN_NAME=admin \
		SEED_RBAC_POLICIES=true \
		SEED_TEST_DATA=true \
		GOVERNMENT_FUNDING_SEED_PATH=configs/government-fundings/berlin.yaml \
		GOVERNMENT_FUNDING_SEED_STATE=berlin \
		CORS_ALLOW_ORIGINS="http://localhost:3000,http://localhost:3001,http://localhost:8080,http://10.149.53.39:3000,http://10.149.53.39:8080" \
		CORS_ALLOW_CREDENTIALS=true \
		LOGIN_RATE_LIMIT_PER_MINUTE=0 \
		API_RATE_LIMIT_PER_MINUTE=0 \
		LOG_FORMAT=text \
		./bin/kitamanager-api > /tmp/kitamanager-api.log 2>&1 & echo $$! > /tmp/kitamanager-api.pid
	@echo "   Waiting for API to be healthy..."
	@for i in $$(seq 1 30); do \
		if curl -sf http://localhost:8080/api/v1/health > /dev/null 2>&1; then \
			echo "   API is ready!"; \
			exit 0; \
		fi; \
		if ! kill -0 $$(cat /tmp/kitamanager-api.pid) 2>/dev/null; then \
			echo "   API process exited before becoming healthy. Last log lines:"; \
			tail -n 20 /tmp/kitamanager-api.log; \
			exit 1; \
		fi; \
		sleep 1; \
	done; \
	echo "   API did not become healthy within 30s. Last log lines:"; \
	tail -n 20 /tmp/kitamanager-api.log; \
	exit 1
	@echo "4. Starting web dev server (Ctrl+C to stop all)..."
	@echo ""
	@echo "================================================"
	@echo "  Web UI: http://localhost:3000 (hot reload)"
	@echo "  API:    http://localhost:8080"
	@echo "  Login:  admin@example.com / supersecret"
	@echo "================================================"
	@echo ""
	@trap 'kill $$(cat /tmp/kitamanager-api.pid) 2>/dev/null; docker compose stop db' EXIT; \
		cd frontend && npx next dev -H 0.0.0.0

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

# Run API unit tests. Race detection roughly doubles wall-clock time and
# the unit suites for models/services/handlers don't exercise real
# concurrency — the packages where data races can actually appear
# (middleware, integration) have a dedicated `api-test-race` target.
api-test-unit:
	go test -v ./...

# Run API integration tests (requires database). Keeps -race because the
# integration suite spins up real HTTP handlers and exercises concurrent
# request paths where -race is load-bearing.
api-test-integration:
	go test -v -race -tags=integration ./internal/integration/...

# Run database backup/restore test (requires Docker)
api-test-backup:
	go test -v -tags=integration -timeout=120s ./scripts/...

# Run API contract tests (requires database)
api-test-contract:
	go test -v -tags=contract ./internal/contract/...

# Run race detector against the packages that actually exercise
# concurrency. Run this in a dedicated CI job (or locally before
# touching middleware/integration code) — running -race across the
# whole tree just doubles per-test cost without finding anything new.
api-test-race:
	go test -v -race ./internal/middleware/...
	go test -v -race -tags=integration ./internal/integration/...

# Run API fuzz tests (each fuzz test must be run separately).
# Use iteration count instead of duration to avoid a known Go fuzz engine race
# condition where time-based context deadlines cause spurious FAIL on slow CI
# runners (golang/go#72104).
api-test-fuzz:
	go test -fuzz=FuzzPeriodOverlaps -fuzztime=1000000x ./internal/models/...
	go test -fuzz=FuzzPeriodIsActiveOn -fuzztime=1000000x ./internal/models/...
	go test -fuzz=FuzzFundingAgeOnDate -fuzztime=1000000x ./internal/validation/...
	go test -fuzz=FuzzEmployeeMonthlyCost -fuzztime=100000x ./internal/service/...

# Run API tests with coverage report. -race is intentionally NOT set:
# coverage is the bottleneck job in CI and doubling its wall-clock for
# race detection (which the unit suites don't surface anyway) was the
# wrong trade. -covermode=count is faster than atomic and is correct
# without -race.
api-test-coverage:
	go test -v -coverprofile=coverage.out -covermode=count ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# =============================================================================
# Web targets (Next.js frontend)
# =============================================================================

# Install web dependencies. Always reinstalls — use this when you want a clean
# node_modules. Targets that merely *need* deps present depend on the
# frontend/node_modules sentinel below instead, so they don't wipe and reinstall
# on every lint.
web-install:
	cd frontend && npm ci

# Sentinel: install only when the lockfile is newer than the installed tree.
# Every npm/npx-driven target below depends on this, because previously a fresh
# clone failed on `make lint`, `make test` and therefore `make ci` — the web
# halves shell out to npm and nothing had installed it. `touch` is needed
# because npm does not reliably bump the directory mtime.
frontend/node_modules: frontend/package-lock.json
	cd frontend && npm ci
	@touch frontend/node_modules

# Start web dev server
web-dev: frontend/node_modules
	cd frontend && npm run dev

# Build web for production
web-build: frontend/node_modules
	cd frontend && npm run build

# Lint web code
web-lint: frontend/node_modules
	cd frontend && npm run lint

# Format web code (Prettier)
web-format: frontend/node_modules
	cd frontend && npm run format

# Check formatting without writing
web-format-check: frontend/node_modules
	cd frontend && npm run format:check

# Type-check web code
web-type-check: frontend/node_modules
	cd frontend && npm run type-check

# Run web unit tests
web-test: frontend/node_modules
	cd frontend && npm run test

# Run web tests with coverage
web-test-coverage: frontend/node_modules
	cd frontend && npm run test:coverage

# Run web E2E tests (requires dev server running or will start it)
web-test-e2e: frontend/node_modules
	cd frontend && npm run test:e2e

# Run web E2E tests with a fresh database (resets all data first)
web-test-e2e-fresh: frontend/node_modules
	@echo "Stopping any running dev environment..."
	@-kill $$(cat /tmp/kitamanager-api.pid 2>/dev/null) 2>/dev/null || true
	@rm -f /tmp/kitamanager-api.pid
	# The frontend has to go too. Playwright's webServer uses
	# reuseExistingServer and keys on :3000, so a surviving `next dev` made it
	# skip starting `make dev` entirely — leaving no API and the empty database
	# we just created, and every test failing for the wrong reason.
	@-pkill -f "next dev" 2>/dev/null || true
	@for i in $$(seq 1 20); do \
		curl -sf http://localhost:3000 >/dev/null 2>&1 || break; \
		sleep 0.5; \
	done
	@curl -sf http://localhost:3000 >/dev/null 2>&1 && \
		{ echo "Something is still serving :3000 — stop it before running this target."; exit 1; } || true
	docker compose down -v
	@echo "Database reset. Playwright will start a fresh dev environment..."
	cd frontend && npm run test:e2e

# Run web E2E tests with browser visible
web-test-e2e-headed: frontend/node_modules
	cd frontend && npm run test:e2e:headed

# Run web E2E tests with browser visible and slow motion (for human watching)
# Uses Chromium for better video recording support
web-test-e2e-demo: frontend/node_modules
	cd frontend && SLOWMO=500 VIDEO=1 npx playwright test --headed --project=chromium

# Install Playwright browsers
web-playwright-install: frontend/node_modules
	cd frontend && npx playwright install --with-deps

# Recapture every website screenshot (all 44, in both en and de).
#
# Needs a running seeded dev environment — start `make dev` in another
# terminal first. The script logs in as the seed admin, walks each screen at
# :3000, and overwrites website/static/images/screenshots/{en,de}/.
#
# tsx is fetched on demand by npx and is deliberately not a frontend
# dependency, so the first run may pause to download it.
#
# The screenshots carry a version watermark, so recapture after any UI change
# that alters labels or layout — otherwise the docs cite a build nobody runs.
screenshots:
	@test -d frontend/node_modules/@playwright/test || \
		{ echo "frontend deps missing — run 'make web-install' first (the script imports @playwright/test)."; exit 1; }
	@curl -sf http://localhost:8080/api/v1/health >/dev/null 2>&1 || \
		{ echo "API not reachable on :8080 — run 'make dev' in another terminal first."; exit 1; }
	@curl -sf http://localhost:3000 >/dev/null 2>&1 || \
		{ echo "Frontend not reachable on :3000 — run 'make dev' in another terminal first."; exit 1; }
	# NODE_PATH is required, not decorative: the script lives in website/scripts/,
	# and Node resolves its imports by walking up from the *script's* directory —
	# website/scripts/, website/, then the repo root — none of which contain
	# node_modules. Without this it dies on "Cannot find module
	# '@playwright/test'" even though the guard above just confirmed the package
	# is installed under frontend/.
	cd frontend && NODE_PATH="$$PWD/node_modules" npx tsx ../website/scripts/capture-screenshots.ts

# =============================================================================
# Documentation targets
# =============================================================================

# Generate OpenAPI/Swagger documentation.
# Pipeline:
#   1. swaggo emits OpenAPI 2.0 → docs/swagger.json (its native format)
#   2. tools/openapi-fixer converts 2.0 → 3.0 → docs/openapi.json
# docs/openapi.json is the canonical contract consumed by the frontend
# type generator (frontend `npm run gen:api`).
swagger-docs:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
	go run ./tools/openapi-fixer/

# Verify the committed spec matches the current Go source. Used in CI.
swagger-check:
	@tmp=$$(mktemp -d) ; \
	swag init -g cmd/api/main.go -o $$tmp --parseDependency --parseInternal >/dev/null ; \
	go run ./tools/openapi-fixer/ -in $$tmp/swagger.json -out $$tmp/openapi.json ; \
	diff -q docs/swagger.json $$tmp/swagger.json >/dev/null 2>&1 || \
	    { echo "docs/swagger.json is stale; run make swagger-docs"; rm -rf $$tmp; exit 1; } ; \
	diff -q docs/openapi.json $$tmp/openapi.json >/dev/null 2>&1 || \
	    { echo "docs/openapi.json is stale; run make swagger-docs"; rm -rf $$tmp; exit 1; } ; \
	rm -rf $$tmp ; \
	echo "spec is current"

# Generate the frontend's TypeScript types from docs/openapi.json. The
# generated file (frontend/src/lib/api/generated.ts) is committed so PR
# diffs make backend type changes visible to reviewers.
api-types: frontend/node_modules
	cd frontend && npm run gen:api

# Verify the committed generated types match the current spec. Used in CI.
api-types-check: frontend/node_modules
	@cd frontend && npm run gen:api >/dev/null
	@if ! git diff --exit-code frontend/src/lib/api/generated.ts >/dev/null 2>&1 ; then \
	    echo "frontend/src/lib/api/generated.ts is stale; run make api-types"; \
	    git diff --stat frontend/src/lib/api/generated.ts; \
	    exit 1; \
	fi
	@echo "generated types are current"

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

# =============================================================================
# Report PDF tool
# =============================================================================

# Build the report-pdf CLI tool. The -ldflags injection mirrors the API
# build (same GitVersion / GitCommit / BuildTime triple) so the tool's
# `--version` flag and the colophon stamped onto every generated PDF
# carry real provenance instead of "dev".
REPORT_VERSION_PKG := github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/version
REPORT_LDFLAGS := -ldflags "-X $(REPORT_VERSION_PKG).GitVersion=$(GIT_VERSION) -X $(REPORT_VERSION_PKG).GitCommit=$(GIT_COMMIT) -X $(REPORT_VERSION_PKG).BuildTime=$(BUILD_TIME)"

report-pdf-build:
	cd tools/report-pdf && go build $(REPORT_LDFLAGS) -o ../../bin/report-pdf .

# Generate PDF reports (requires running dev environment).
# Depends on the build: without it this ran ./bin/report-pdf on a clean tree and
# failed with "no such file".
report-pdf: report-pdf-build
	./bin/report-pdf --email admin@example.com --password supersecret --org-id 1
