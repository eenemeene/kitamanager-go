---
title: Make targets
weight: 3
---

The repo's `Makefile` is the canonical entry point for build, test, and tooling commands. Targets are grouped here by purpose.

## Combined

| Target | What it does |
|---|---|
| `make build` | Build both web and API. |
| `make lint` | Lint both web and API. |
| `make test` | Run both web and API unit tests. |
| `make ci` | Combined lint + build + test (mirrors what CI runs). |
| `make clean` | Remove `bin/`, the coverage report, and the Next.js build cache. |
| `make dev` | Start the full development stack (Postgres + API + web dev server). |
| `make dev-fresh` | Same as `dev` but resets the database first. |

## API (Go)

| Target | What it does |
|---|---|
| `make api-build` | Build the `kitamanager-api` binary into `bin/`. |
| `make api-run` | Build and run the API. |
| `make api-lint` | Run `golangci-lint`. |
| `make api-test-all` | Run unit + integration + contract tests. |
| `make api-test-unit` | Run unit tests for every package. |
| `make api-test-integration` | Run integration tests (`-tags=integration`, requires Postgres). |
| `make api-test-contract` | Run API contract tests (`-tags=contract`). |
| `make api-test-fuzz` | Run the fuzz suites. |
| `make api-test-coverage` | Run unit tests with coverage profile. |
| `make api-test-backup` | Run the DB backup/restore integration test (requires Docker). |
| `make api-test-race` | Run `-race` against the packages where concurrency actually lives (`internal/middleware`, `internal/integration`). |

## Web (Next.js)

| Target | What it does |
|---|---|
| `make web-install` | `npm ci` in `frontend/`. |
| `make web-dev` | Run the Next.js dev server. |
| `make web-build` | Build the production bundle. |
| `make web-lint` | Run ESLint. |
| `make web-format` | Run Prettier writeback. |
| `make web-format-check` | Check Prettier formatting (CI). |
| `make web-type-check` | Run `tsc --noEmit`. |
| `make web-test` | Run Jest tests. |
| `make web-test-coverage` | Run Jest with coverage. |
| `make web-test-e2e` | Run Playwright tests against the running stack. |
| `make web-test-e2e-fresh` | Same as `web-test-e2e` but resets the DB first. |
| `make web-test-e2e-headed` | Same as `web-test-e2e` but in a headed browser for live debugging. |
| `make web-test-e2e-demo` | Playwright in headed mode with slow motion + video recording for demos. |
| `make web-playwright-install` | Install Playwright browsers. |
| `make screenshots` | Recapture all 44 website screenshots in both languages into `website/static/images/screenshots/{en,de}/`. Needs `make dev` running in another terminal; fails fast with a message if the API, the frontend, or the frontend deps are missing. |

## Documentation

| Target | What it does |
|---|---|
| `make docs` | Regenerates the auto-built docs: swagger and the schema diagram. (Does **not** run Hugo — for the website, run `hugo --source=website` directly or use `hugo --source=website server` for live preview.) |
| `make schema-docs` | Regenerates `docs/schema/` from the running database via `tbls`. |
| `make swagger-docs` | Regenerates `docs/swagger.{json,yaml}` from swaggo annotations. |
| `make swagger-check` | Verifies the swagger files are up to date (CI). |
| `make api-types` | Regenerates the TypeScript API types from the OpenAPI spec. |
| `make api-types-check` | Verifies the generated types are up to date (CI). |

## Docker / local infra

| Target | What it does |
|---|---|
| `make docker-up` | `docker compose up -d`. |
| `make docker-down` | `docker compose down`. |
| `make docker-rebuild` | Rebuild images and restart (`docker compose up -d --build` — the layer cache is reused; pass `--no-cache` manually for a clean build). |
| `make docker-reset` | Stop, drop volumes, restart. |

## Git hooks

| Target | What it does |
|---|---|
| `make install-hooks` | Install pre-commit hooks. |
| `make uninstall-hooks` | Remove pre-commit hooks. |
| `make pre-commit` | Run all pre-commit checks once. |

## Report PDF tool

| Target | What it does |
|---|---|
| `make report-pdf-build` | Build the `report-pdf` binary into `bin/report-pdf`. |
| `make report-pdf` | Run the previously-built report tool against a configured stack (uses the demo `admin@example.com` / `supersecret` credentials by default). |
