---
title: Set up a development environment
weight: 1
---

You want to run KitaManager locally with hot-reload for both the Go API and the Next.js frontend.

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 24+](https://nodejs.org/)
- [Docker](https://docs.docker.com/get-docker/) (for Postgres in dev)
- `make`

## Steps

```bash
# 1. Install frontend dependencies
make web-install

# 2. Build the API binary (also generates types)
make api-build

# 3. Start the full dev stack: Postgres + API + frontend
make dev
```

The first run takes a couple of minutes (Docker pulls, Go module download, npm install, initial build). Subsequent `make dev` starts in seconds.

## What you get

- Postgres on `localhost:5432` (credentials in `.env.dev.example`).
- API on `http://localhost:8080`. OpenAPI UI at `http://localhost:8080/swagger/index.html`.
- Frontend on `http://localhost:3000`.
- Demo accounts seeded automatically (`superadmin@example.com`, `admin@example.com`, `manager@example.com` — all with password `supersecret`).

## Reset the dev DB

```bash
make dev-fresh
```

drops the database, re-runs migrations, and re-seeds. Use this when a schema change makes your local data inconsistent or when you want a clean slate.

## Notes

- For the full Make-target list, see [Make targets](../../../reference/cli-and-config/make-targets/).
- For the env var list, see [Environment variables](../../../reference/cli-and-config/env-vars/) — the dev defaults live in `.env.dev.example`.
- The pre-commit hooks run a fairly strict suite (`golangci-lint`, `go test`, web type-check, web lint, prettier). Install them with `make install-hooks`.
- The repo's `CLAUDE.md` and `.claude/rules/*.md` files are the source of truth for code-level conventions; read them before opening a PR.
