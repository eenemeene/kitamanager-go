---
title: Mit Docker Compose bereitstellen
weight: 1
---

Sie wollen KitaManager (Postgres + API + Frontend) auf einem Host mit Docker Compose hochfahren. Das ist der gleiche Pfad wie das Tutorial [KitaManager bereitstellen](../../../tutorials/deploy-kitamanager/), nur als kompakteres Rezept für jemanden, der schon weiß, was zu tun ist.

## Voraussetzungen

- Docker und Docker Compose installiert.
- Offene Ports für Frontend und API (Standard `3000` und `8080`).
- Ein langes zufälliges `CSRF_HMAC_KEY` (mindestens 32 Zeichen) und ein 64-Zeichen-Hex-`TOTP_ENCRYPTION_KEY`.

## Schritte

1. Repository auschecken auf der gewünschten Version.
2. `.env.production.example` nach `.env` kopieren und ausfüllen:
   - `DB_USER`, `DB_PASSWORD`, `DB_NAME`
   - `CSRF_HMAC_KEY` — `openssl rand -hex 32`
   - `TOTP_ENCRYPTION_KEY` — `openssl rand -hex 32`
   - `WEBAUTHN_RP_ID`, `WEBAUTHN_RP_NAME`, `WEBAUTHN_ORIGINS`, falls Sicherheitsschlüssel genutzt werden
   - `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`, `SEED_ADMIN_NAME` (für den Bootstrap-Superadmin)
   - `GOVERNMENT_FUNDING_SEED_PATH=configs/government-fundings/berlin.yaml`, um Berliner Sätze beim ersten Start zu laden
3. Hochfahren:
   ```bash
   docker compose up -d
   ```
4. Auf den Health-Check der API warten:
   ```bash
   curl -sf http://localhost:8080/api/v1/health
   ```
5. Bei `http://localhost:3000` mit Ihrer `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` anmelden. Passwort sofort ändern, 2FA aktivieren.

## Hinweise

- Vollständige Env-Var-Referenz: [Umgebungsvariablen](../../../reference/cli-and-config/env-vars/).
- Für den *erstmaligen* Durchlauf mit Beispieldaten folgen Sie stattdessen dem Tutorial [KitaManager bereitstellen](../../../tutorials/deploy-kitamanager/).
- Produktiv-Checkliste:
  - `SECURE_COOKIES=true`
  - `DB_SSLMODE=require` (oder strenger)
  - `LOGIN_RATE_LIMIT_PER_MINUTE` auf Default (5) und `API_RATE_LIMIT_PER_MINUTE` (60), oder an Ihren Traffic angepasst
  - **Niemals** `SEED_TEST_DATA=true` in Produktion setzen
  - Sofort das erste Backup machen — siehe [Datenbank sichern und wiederherstellen](../back-up-and-restore/)
