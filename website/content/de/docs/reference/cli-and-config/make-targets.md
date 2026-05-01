---
title: Make-Ziele
weight: 3
---

Das `Makefile` des Repositories ist der maßgebliche Einstiegspunkt für Build, Tests und Tooling. Die Ziele sind hier nach Zweck gruppiert.

## Kombiniert

| Ziel | Was es tut |
|---|---|
| `make build` | Baut sowohl Web als auch API. |
| `make lint` | Lintet sowohl Web als auch API. |
| `make test` | Führt Web- und API-Unit-Tests aus. |
| `make ci` | Kombiniertes Lint + Build + Test (entspricht dem CI-Lauf). |
| `make dev` | Startet den vollständigen Entwicklungs-Stack (Postgres + API + Web-Dev-Server). |
| `make dev-fresh` | Wie `dev`, setzt aber zuerst die Datenbank zurück. |

## API (Go)

| Ziel | Was es tut |
|---|---|
| `make api-build` | Baut die `kitamanager-api`-Binärdatei nach `bin/`. |
| `make api-run` | Baut und startet die API. |
| `make api-lint` | Führt `golangci-lint` aus. |
| `make api-test-all` | Unit + Integration + Contract-Tests. |
| `make api-test-unit` | Unit-Tests für jedes Paket. |
| `make api-test-integration` | Integrationstests (`-tags=integration`, erfordert Postgres). |
| `make api-test-contract` | API-Contract-Tests (`-tags=contract`). |
| `make api-test-fuzz` | Führt die Fuzz-Suiten aus. |
| `make api-test-coverage` | Unit-Tests mit Coverage-Profil. |
| `make api-test-backup` | DB-Backup/Restore-Integrationstest (erfordert Docker). |
| `make api-test-race` | `-race` gegen die Pakete, in denen Concurrency tatsächlich existiert (`internal/middleware`, `internal/integration`). |

## Web (Next.js)

| Ziel | Was es tut |
|---|---|
| `make web-install` | `npm ci` in `frontend/`. |
| `make web-dev` | Startet den Next.js-Dev-Server. |
| `make web-build` | Baut das Produktions-Bundle. |
| `make web-lint` | Führt ESLint aus. |
| `make web-format` | Führt Prettier-Writeback aus. |
| `make web-format-check` | Prüft Prettier-Formatierung (CI). |
| `make web-type-check` | Führt `tsc --noEmit` aus. |
| `make web-test` | Führt Jest-Tests aus. |
| `make web-test-coverage` | Führt Jest mit Coverage aus. |
| `make web-test-e2e` | Führt Playwright-Tests gegen den laufenden Stack aus. |
| `make web-test-e2e-fresh` | Wie `web-test-e2e`, setzt aber zuerst die DB zurück. |
| `make web-test-e2e-demo` | Playwright im Headed-Modus zur manuellen Inspektion. |
| `make web-playwright-install` | Installiert Playwright-Browser. |

## Dokumentation

| Ziel | Was es tut |
|---|---|
| `make docs` | Regeneriert die auto-gebauten Docs: swagger und das Schema-Diagramm. (Führt **nicht** Hugo aus — für die Website direkt `hugo --source=website` oder `hugo --source=website server` für Live-Preview nutzen.) |
| `make schema-docs` | Regeneriert `docs/schema/` aus der laufenden Datenbank über `tbls`. |
| `make swagger-docs` | Regeneriert `docs/swagger.{json,yaml}` aus den swaggo-Annotationen. |
| `make swagger-check` | Verifiziert, dass die swagger-Dateien aktuell sind (CI). |
| `make api-types` | Regeneriert die TypeScript-API-Typen aus der OpenAPI-Spec. |
| `make api-types-check` | Verifiziert, dass die generierten Typen aktuell sind (CI). |

## Docker / lokale Infrastruktur

| Ziel | Was es tut |
|---|---|
| `make docker-up` | `docker compose up -d`. |
| `make docker-down` | `docker compose down`. |
| `make docker-rebuild` | Baut Images ohne Cache neu und startet neu. |
| `make docker-reset` | Stoppt, droppt Volumes, startet neu. |

## Git-Hooks

| Ziel | Was es tut |
|---|---|
| `make install-hooks` | Pre-Commit-Hooks installieren. |
| `make uninstall-hooks` | Pre-Commit-Hooks entfernen. |
| `make pre-commit` | Alle Pre-Commit-Checks einmal ausführen. |

## Report-PDF-Tool

| Ziel | Was es tut |
|---|---|
| `make report-pdf-build` | Baut die `kitamanager-report`-Binärdatei. |
| `make report-pdf` | Baut das Report-Tool und startet es gegen einen konfigurierten Stack. |
