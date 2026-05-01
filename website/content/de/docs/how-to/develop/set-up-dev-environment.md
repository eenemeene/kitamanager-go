---
title: Entwicklungsumgebung einrichten
weight: 1
---

Sie wollen KitaManager lokal mit Hot-Reload für Go-API und Next.js-Frontend laufen lassen.

## Voraussetzungen

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 24+](https://nodejs.org/)
- [Docker](https://docs.docker.com/get-docker/) (für Postgres in der Entwicklung)
- `make`

## Schritte

```bash
# 1. Frontend-Abhängigkeiten installieren
make web-install

# 2. API-Binary bauen (generiert auch die Typen)
make api-build

# 3. Vollständigen Dev-Stack starten: Postgres + API + Frontend
make dev
```

Der erste Start dauert ein paar Minuten (Docker-Pulls, Go-Module-Download, npm install, initialer Build). Folgende `make dev` starten in Sekunden.

## Was Sie bekommen

- Postgres auf `localhost:5432` (Credentials in `.env.dev.example`).
- API auf `http://localhost:8080`. OpenAPI-UI unter `http://localhost:8080/swagger/index.html`.
- Frontend auf `http://localhost:3000`.
- Demo-Konten automatisch geseedet (`superadmin@example.com`, `admin@example.com`, `manager@example.com` — alle mit Passwort `supersecret`).

## Dev-DB zurücksetzen

```bash
make dev-fresh
```

droppt die Datenbank, führt Migrationen neu aus und seedet neu. Nutzen Sie das, wenn eine Schema-Änderung Ihre lokalen Daten inkonsistent macht oder wenn Sie einen sauberen Start wollen.

## Hinweise

- Vollständige Make-Ziel-Liste: [Make-Ziele](../../../reference/cli-and-config/make-targets/).
- Vollständige Env-Var-Liste: [Umgebungsvariablen](../../../reference/cli-and-config/env-vars/) — die Dev-Defaults liegen in `.env.dev.example`.
- Die Pre-Commit-Hooks führen eine recht strenge Suite aus (`golangci-lint`, `go test`, web type-check, web lint, prettier). Installieren mit `make install-hooks`.
- Die Repo-Dateien `CLAUDE.md` und `.claude/rules/*.md` sind die maßgeblichen Quellen für Code-Konventionen; vor einem PR lesen.
